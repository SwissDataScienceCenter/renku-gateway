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
"""Implement Keycloak authentication workflow for CLI."""

import json
import re
import time
from base64 import b64encode
from urllib.parse import urljoin

from flask import (
    Blueprint,
    current_app,
    redirect,
    render_template,
    request,
    session,
    url_for,
)

from .gitlab_auth import GL_SUFFIX, SCOPE
from .oauth_provider_app import GitLabProviderApp
from .utils import (
    get_redis_key_from_token,
    handle_login_request,
    handle_token_request, get_redis_key_for_cli, decode_keycloak_jwt, get_redis_key_from_session,
)

CLI_SUFFIX = "cli_oauth_client"

blueprint = Blueprint("cli_auth", __name__, url_prefix="/auth/cli")


class RenkuCoreCLIAuthHeaders:
    def process(self, request, headers):

        m = re.search(
            r"bearer (?P<token>.+)", headers.get("Authorization", ""), re.IGNORECASE
        )
        if m:
            access_token = m.group("token")

            keycloak_oidc_client = current_app.store.get_oauth_client(
                get_redis_key_from_token(access_token, key_suffix=CLI_SUFFIX)
            )
            headers["Renku-User"] = keycloak_oidc_client.token["id_token"]

            gitlab_oauth_client = current_app.store.get_oauth_client(
                get_redis_key_from_token(access_token, key_suffix=GL_SUFFIX)
            )
            headers["Authorization"] = "Bearer {}".format(
                gitlab_oauth_client.access_token
            )

            # TODO: The subsequent information is now included in the JWT sent in the
            # TODO: "Renku-User" header. It can be removed once the core-service
            # TODO: relies on the new header.
            access_token_dict = decode_keycloak_jwt(access_token.encode())
            headers["Renku-user-id"] = access_token_dict["sub"]
            headers["Renku-user-email"] = b64encode(access_token_dict["email"].encode())
            headers["Renku-user-fullname"] = b64encode(
                access_token_dict["name"].encode()
            )

        else:
            pass

        return headers


@blueprint.route("/login")
def login():
    current_app.logger.warn("LOG: cli_auth.login called")
    # provider_app = GitLabProviderApp(
    #     client_id=current_app.config["GITLAB_CLIENT_ID"],
    #     client_secret=current_app.config["GITLAB_CLIENT_SECRET"],
    #     base_url=current_app.config["GITLAB_URL"],
    # )
    # return handle_login_request(
    #     provider_app,
    #     urljoin(current_app.config["HOST_NAME"], url_for("cli_auth.token")),
    #     CLI_SUFFIX,
    #     SCOPE,
    # )
    gitlab_redis_key = get_redis_key_from_session(key_suffix=GL_SUFFIX)
    gitlab_oauth_client = current_app.store.get_oauth_client(gitlab_redis_key)

    redis_key = get_redis_key_from_session(key_suffix=CLI_SUFFIX)
    current_app.store.set_oauth_client(redis_key, gitlab_oauth_client)
    # current_app.logger.warn(f"LOG: HANDLING LOGIN {redirect_path} {authorization_url}")
    token_url = urljoin(current_app.config["HOST_NAME"], url_for("cli_auth.token"))

    return current_app.make_response(redirect(token_url))


@blueprint.route("/token")
def token():
    current_app.logger.warn("LOG: cli_auth.token called")
    # response, _ = handle_token_request(request, CLI_SUFFIX)

    client_redis_key = get_redis_key_from_session(key_suffix=CLI_SUFFIX)
    cli_nonce = session.get("cli_nonce")
    if cli_nonce:
        server_nonce = session.get("server_nonce")
        cli_redis_key = get_redis_key_for_cli(cli_nonce, server_nonce)
        login_info = CLILoginInfo(client_redis_key)
        current_app.store.set_enc(cli_redis_key, login_info.to_json().encode())
    else:
        current_app.logger.warn("cli_auth.token called without cli_nonce")

    response = current_app.make_response(
        redirect(
            urljoin(current_app.config["HOST_NAME"], url_for("web_auth.login_next"))
        )
    )
    return response


@blueprint.route("/logout")
def logout():
    return ""


class CLILoginInfo:
    """Stores some information about CLI login."""

    def __init__(self, client_redis_key, login_start=None):
        self.client_redis_key = client_redis_key
        self.login_start = login_start or time.time()

    @classmethod
    def from_json(cls, json_string):
        """Create an instance from json string."""
        data = json.loads(json_string)
        return cls(**data)

    def to_json(self):
        """Serialize an instance to json string."""
        data = {
            "client_redis_key": self.client_redis_key, "login_start": self.login_start
        }
        return json.dumps(data)

    def is_expired(self):
        elapsed = time.time() - self.login_start
        return elapsed > current_app.config["CLI_LOGIN_TIMEOUT"]


def handle_cli_token_request(request):
    """Handle final stage of CLI login."""
    cli_nonce = request.args.get("cli_nonce")
    server_nonce = request.args.get("server_nonce")
    if not cli_nonce or not server_nonce:
        return {"error": "Required arguments are missing."}, 400

    cli_redis_key = get_redis_key_for_cli(cli_nonce, server_nonce)
    current_app.logger.debug(f"Looking up Keycloak for CLI login request {cli_nonce}")
    data = current_app.store.get_enc(cli_redis_key)
    if not data:
        return {"error": "Something went wrong internally."}, 500
    current_app.store.delete(cli_redis_key)

    login_info = CLILoginInfo.from_json(data.decode())
    if login_info.is_expired():
        return {"error": "Session expired."}, 403

    oauth_client = current_app.store.get_oauth_client(
        login_info.client_redis_key, no_refresh=True
    )
    if not oauth_client:
        return {"error": "Access token not found."}, 404

    return {"access_token": oauth_client.access_token}
