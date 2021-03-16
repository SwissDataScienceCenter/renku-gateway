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
from urllib.parse import urljoin

from flask import Blueprint, current_app, request, session, url_for

from .gitlab_auth import GL_SUFFIX
from .oauth_provider_app import KeycloakProviderApp
from .utils import (
    get_redis_key_for_cli,
    get_redis_key_from_session,
    get_redis_key_from_token,
    handle_login_request,
    handle_token_request,
)

blueprint = Blueprint("cli_auth", __name__, url_prefix="/auth/cli")

CLI_SUFFIX = "cli_oauth_client"
SCOPE = ["openid"]


class RenkuCoreCLIAuthHeaders:
    def process(self, request, headers):
        authorization = headers.get("Authorization")
        authorization_match = (
            re.search(r"bearer\s+(?P<token>.+)", authorization, re.IGNORECASE)
            if authorization
            else None
        )
        if authorization_match:
            token = authorization_match.group("token")
            audience = current_app.config["CLI_CLIENT_ID"]
            redis_key = get_redis_key_from_token(
                token, key_suffix=GL_SUFFIX, audience=audience
            )
            gitlab_oauth_client = current_app.store.get_oauth_client(redis_key)
            headers["Authorization"] = f"Bearer {gitlab_oauth_client.access_token}"

        return headers


@blueprint.route("/login")
def login():
    provider_app = KeycloakProviderApp(
        client_id=current_app.config["CLI_CLIENT_ID"],
        client_secret=current_app.config["CLI_CLIENT_SECRET"],
        base_url=current_app.config["OIDC_ISSUER"],
    )
    return handle_login_request(
        provider_app,
        urljoin(current_app.config["HOST_NAME"], url_for("cli_auth.token")),
        CLI_SUFFIX,
        SCOPE,
    )


@blueprint.route("/token")
def token():
    response, _ = handle_token_request(request, CLI_SUFFIX)

    client_redis_key = get_redis_key_from_session(key_suffix=CLI_SUFFIX)
    cli_nonce = session.get("cli_nonce")
    if cli_nonce:
        server_nonce = session.get("server_nonce")
        cli_redis_key = get_redis_key_for_cli(cli_nonce, server_nonce)
        login_info = CLILoginInfo(client_redis_key)
        current_app.store.set_enc(cli_redis_key, login_info.to_json().encode())
    else:
        current_app.logger.warn("cli_auth.token called without cli_nonce")

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
            "client_redis_key": self.client_redis_key,
            "login_start": self.login_start,
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
