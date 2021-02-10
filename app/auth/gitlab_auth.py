# -*- coding: utf-8 -*-
#
# Copyright 2018 - Swiss Data Science Center (SDSC)
# A partnership between École Polytechnique Fédérale de Lausanne (EPFL) and
# Eidgenössische Technische Hochschule Zürich (ETHZ).
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Implement GitLab authentication workflow."""

import base64
import re
from urllib.parse import urljoin

from flask import (
    Blueprint,
    current_app,
    redirect,
    render_template,
    request,
    url_for,
)

from .oauth_provider_app import GitLabProviderApp
from .utils import (
    get_redis_key_from_token,
    handle_login_request,
    handle_token_request,
)

GL_SUFFIX = "gl_oauth_client"

blueprint = Blueprint("gitlab_auth", __name__, url_prefix="/auth/gitlab")


# Note: GitLab oauth tokens do NOT expire per default
# See https://gitlab.com/gitlab-org/gitlab/-/issues/21745
# The documentation about this is wrong
# (https://docs.gitlab.com/ce/api/oauth2.html#web-application-flow)


class GitlabUserToken:
    def process(self, request, headers):
        m = re.search(
            r"bearer (?P<token>.+)", headers.get("Authorization", ""), re.IGNORECASE
        )
        if m:
            # current_app.logger.debug(
            #    'outgoing headers: {}'.format(json.dumps(headers)
            # )
            access_token = m.group("token")
            redis_key = get_redis_key_from_token(access_token, key_suffix=GL_SUFFIX)
            gitlab_oauth_client = current_app.store.get_oauth_client(redis_key)

            if gitlab_oauth_client:
                renku_token = headers.pop("Renku-Token", None)
                gitlab_access_token = gitlab_oauth_client.access_token

                if renku_token:  # Request comes from git CLI; create basic auth header
                    user_pass = f"oauth2:{gitlab_access_token}".encode("utf-8")
                    basic_auth = base64.b64encode(user_pass).decode("ascii")
                    headers["Authorization"] = f"Basic {basic_auth}"
                else:
                    headers["Authorization"] = f"Bearer {gitlab_access_token}"
        else:
            current_app.logger.debug(
                "No authorization header, returning empty auth headers"
            )

        return headers


SCOPE = ["openid", "api", "read_user", "read_repository"]


@blueprint.route("/login")
def login():
    provider_app = GitLabProviderApp(
        client_id=current_app.config["GITLAB_CLIENT_ID"],
        client_secret=current_app.config["GITLAB_CLIENT_SECRET"],
        base_url=current_app.config["GITLAB_URL"],
    )
    return handle_login_request(
        provider_app,
        urljoin(current_app.config["HOST_NAME"], url_for("gitlab_auth.token")),
        GL_SUFFIX,
        SCOPE,
    )


@blueprint.route("/token")
def token():
    response, _ = handle_token_request(request, GL_SUFFIX)
    return response


@blueprint.route("/logout")
def logout():
    logout_url = current_app.config["GITLAB_URL"] + "/users/sign_out"

    # For gitlab versions previous to 12.7.0 we need to redirect the
    # browser to the logout url. For versions 12.9.0 and newer the
    # browser has to POST a form to the logout url.
    if current_app.config["OLD_GITLAB_LOGOUT"]:
        response = current_app.make_response(redirect(logout_url))
    else:
        response = render_template("gitlab_logout.html", logout_url=logout_url)

    return response
