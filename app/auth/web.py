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
"""Web auth routes."""

import re
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

from .oauth_provider_app import KeycloakProviderApp
from .gitlab_auth import GL_SUFFIX
from .jupyterhub_auth import JH_SUFFIX
from .utils import (
    decode_keycloak_jwt,
    get_redis_key_from_cli_token,
    get_redis_key_from_session,
    handle_login_request,
    TEMP_SESSION_KEY,
    handle_token_request,
    get_redis_key_from_refresh_token,
    verify_refresh_token,
)

blueprint = Blueprint("web_auth", __name__, url_prefix="/auth")

KC_SUFFIX = "kc_oidc_client"
SCOPE = ["openid"]


def get_valid_token(headers):
    """
    Look for a fresh and valid token, first in headers, then in the session.

    If a refresh token is available, it can be swapped for an access token.
    """

    def get_access_token_from_refresh_token(refresh_token):
        redis_key = get_redis_key_from_refresh_token(
            refresh_token, key_suffix=KC_SUFFIX
        )
        oauth_client = current_app.store.get_oauth_client(redis_key)
        if verify_refresh_token(refresh_token, oauth_client):
            return oauth_client.access_token

    renku_token = headers.get("Renku-Token")
    authorization = headers.get("Authorization")
    authorization_match = (
        re.search(r"bearer\s+(?P<token>.+)", authorization, re.IGNORECASE)
        if authorization
        else None
    )

    # If Renku-Token exists it comes from git in CLI and it is a refresh token
    if renku_token:
        return get_access_token_from_refresh_token(renku_token)
    elif authorization_match:  # If token bearer exists it's a refresh token too
        refresh_token = authorization_match.group("token")
        return get_access_token_from_refresh_token(refresh_token)
    elif headers.get("X-Requested-With") == "XMLHttpRequest" and "sub" in session:
        redis_key = get_redis_key_from_session(key_suffix=KC_SUFFIX)
        keycloak_oidc_client = current_app.store.get_oauth_client(redis_key)
        return keycloak_oidc_client.access_token


LOGIN_SEQUENCE = ["web_auth.login", "gitlab_auth.login", "jupyterhub_auth.login"]


@blueprint.route("/login/next")
def login_next():
    session["login_seq"] += 1
    if session["login_seq"] < len(LOGIN_SEQUENCE):
        return render_template(
            "redirect.html",
            redirect_url=urljoin(
                current_app.config["HOST_NAME"],
                url_for(LOGIN_SEQUENCE[session["login_seq"]]),
            ),
        )
    else:
        return redirect(session["ui_redirect_url"])


@blueprint.route("/login")
def login():
    """Log in with Keycloak."""
    session.clear()
    session["ui_redirect_url"] = (
        request.args.get("redirect_url") or current_app.config["HOST_NAME"]
    )
    session["cli_token"] = request.args.get("cli_token")
    session["login_seq"] = 0

    provider_app = KeycloakProviderApp(
        client_id=current_app.config["OIDC_CLIENT_ID"],
        client_secret=current_app.config["OIDC_CLIENT_SECRET"],
        base_url=current_app.config["OIDC_ISSUER"],
    )
    return handle_login_request(
        provider_app,
        urljoin(current_app.config["HOST_NAME"], url_for("web_auth.token")),
        KC_SUFFIX,
        SCOPE,
    )


@blueprint.route("/token")
def token():
    response, keycloak_oidc_client = handle_token_request(request, KC_SUFFIX)

    # Get rid of the temporarily stored oauth client object in redis
    # and the reference to it in the session.
    old_redis_key = get_redis_key_from_session(key_suffix=KC_SUFFIX)
    current_app.store.delete(old_redis_key)
    session.pop(TEMP_SESSION_KEY, None)

    # Store the oauth client object in redis under the regular "sub" claim.
    session["sub"] = decode_keycloak_jwt(keycloak_oidc_client.access_token)["sub"]
    new_redis_key = get_redis_key_from_session(key_suffix=KC_SUFFIX)
    current_app.store.set_oauth_client(new_redis_key, keycloak_oidc_client)

    cli_token = session.get("cli_token")
    if cli_token:
        cli_redis_key = get_redis_key_from_cli_token(cli_token)
        current_app.store.set_oauth_client(cli_redis_key, keycloak_oidc_client)

    return response


@blueprint.route("/info")
def info():
    cli_token = request.args.get("cli_token")
    if not cli_token:
        return {"error": "CLI token is missing."}, 400

    cli_redis_key = get_redis_key_from_cli_token(cli_token)
    current_app.logger.debug(f"Looking up Keycloak for request {cli_token}")
    keycloak_oidc_client = current_app.store.get_oauth_client(
        cli_redis_key, no_refresh=True
    )

    if keycloak_oidc_client:
        current_app.store.delete(cli_redis_key)
        # Note: Send refresh token to CLI
        return {"access_token": keycloak_oidc_client.refresh_token}
    else:
        return {"error": "Access token not found"}, 404


# @blueprint.route("/user")
# async def user():
#     from .. import store

#     if "token" not in session:
#         return await current_app.make_response(
#             redirect(
#                 "{}?redirect_url={}".format(
#                     url_for("web_auth.login"), quote_plus(url_for("web_auth.user"))
#                 )
#             )
#         )
#     try:
#         a = jwt.decode(
#             session["token"],
#             current_app.config["OIDC_PUBLIC_KEY"],
#             algorithms=JWT_ALGORITHM,
#             audience=current_app.config["OIDC_CLIENT_ID"],
#         )  # TODO: logout and redirect if fails because of expired

#         return current_app.store.get(get_redis_key(a, "kc_id_token")).decode()

#     except jwt.ExpiredSignatureError:
#         return await current_app.make_response(
#             redirect(
#                 "{}?redirect_url={}".format(
#                     url_for("web_auth.login"), quote_plus(url_for("web_auth.user"))
#                 )
#             )
#         )


@blueprint.route("/user-profile")
def user_profile():
    return current_app.make_response(
        redirect("{}/account?referrer=renku".format(current_app.config["OIDC_ISSUER"]))
    )


@blueprint.route("/logout")
def logout():

    if "sub" in session:
        current_app.store.delete(get_redis_key_from_session(key_suffix=GL_SUFFIX))
        current_app.store.delete(get_redis_key_from_session(key_suffix=JH_SUFFIX))
        current_app.store.delete(get_redis_key_from_session(key_suffix=KC_SUFFIX))
    session.clear()

    logout_pages = [
        urljoin(current_app.config["HOST_NAME"], url_for("jupyterhub_auth.logout")),
        urljoin(current_app.config["HOST_NAME"], url_for("gitlab_auth.logout")),
        f"{current_app.config['OIDC_ISSUER']}/protocol/openid-connect/logout",
    ]

    return render_template(
        "redirect_logout.html",
        redirect_url=request.args.get("redirect_url"),
        logout_pages=logout_pages,
        len=len(logout_pages),
    )
