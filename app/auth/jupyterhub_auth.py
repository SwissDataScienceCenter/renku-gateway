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
"""Implement JupyterHub authentication workflow."""
import re
from urllib.parse import urlencode, urljoin

from flask import Blueprint, Response, current_app, redirect, request, url_for

from .oauth_provider_app import JupyterHubProviderApp
from .utils import (
    get_redis_key_from_token,
    handle_login_request,
    handle_token_request,
)

JH_SUFFIX = "jh_oauth_client"

blueprint = Blueprint("jupyterhub_auth", __name__, url_prefix="/auth/jupyterhub")


class JupyterhubUserToken:
    def process(self, request, headers):

        m = re.search(
            r"bearer (?P<token>.+)", headers.get("Authorization", ""), re.IGNORECASE
        )
        if m:
            # current_app.logger.debug('Authorization header present, token exchange')
            access_token = m.group("token")
            jupyterhub_oauth_client = current_app.store.get_oauth_client(
                get_redis_key_from_token(access_token, key_suffix=JH_SUFFIX)
            )
            headers["Authorization"] = "token {}".format(
                jupyterhub_oauth_client.access_token
            )

            # current_app.logger.debug(
            #    'outgoing headers: {}'.format(json.dumps(headers))
            # )
        else:
            # current_app.logger.debug(
            #    "No authorization header, returning empty auth headers"
            # )
            headers.pop("Authorization", None)

        return headers


@blueprint.route("/login")
def login():
    provider_app = JupyterHubProviderApp(
        client_id=current_app.config["JUPYTERHUB_CLIENT_ID"],
        client_secret=current_app.config["JUPYTERHUB_CLIENT_SECRET"],
        base_url=current_app.config["JUPYTERHUB_URL"],
    )
    return handle_login_request(
        provider_app,
        urljoin(current_app.config["HOST_NAME"], url_for("jupyterhub_auth.token")),
        JH_SUFFIX,
        [],
    )


@blueprint.route("/login-tmp")
def login_tmp():
    """Redirection creating an anonymous (temporary) Jupyterhub session."""

    if not current_app.config["ANONYMOUS_SESSIONS_ENABLED"]:
        return Response("Anonymous notebooks sessions are disabled", status=404)

    args = {"redirect_url": request.args["redirect_url"]}

    full_url = "{}{}?{}".format(
        current_app.config["JUPYTERHUB_TMP_URL"],
        "/services/notebooks/login-tmp",
        urlencode(args),
    )

    return current_app.make_response(redirect(full_url))


@blueprint.route("/token")
def token():
    response, _ = handle_token_request(request, JH_SUFFIX)
    return response


@blueprint.route("/logout")
def logout():
    logout_url = current_app.config["JUPYTERHUB_URL"] + "/hub/logout"
    return current_app.make_response(redirect(logout_url))
