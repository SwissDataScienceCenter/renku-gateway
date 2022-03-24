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

from urllib.parse import urljoin

from flask import (
    Blueprint,
    current_app,
    render_template,
    request,
    url_for,
)

from ..common.oauth_provider_app import GitLabProviderApp
from ..common.utils import (
    build_redis_key,
    handle_login_request,
    handle_token_request,
)

GL_SUFFIX = "gl_oauth_client"

blueprint = Blueprint("gitlab_auth", __name__, url_prefix="auth/gitlab")


# Note: GitLab oauth tokens do NOT expire per default
# See https://gitlab.com/gitlab-org/gitlab/-/issues/21745
# The documentation about this is wrong
# (https://docs.gitlab.com/ce/api/oauth2.html#web-application-flow)


def add_auth_headers(sub, headers):
    redis_key = build_redis_key(sub, key_suffix=GL_SUFFIX)
    gitlab_oauth_client = current_app.store.get_oauth_client(redis_key)

    if gitlab_oauth_client:
        gitlab_access_token = gitlab_oauth_client.access_token
        headers["Authorization"] = f"Bearer {gitlab_access_token}"
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
    return handle_token_request(request, GL_SUFFIX)


@blueprint.route("/logout")
def logout():
    return render_template(
        "gitlab_logout.html",
        logout_url=current_app.config["GITLAB_URL"] + "/users/sign_out",
    )
