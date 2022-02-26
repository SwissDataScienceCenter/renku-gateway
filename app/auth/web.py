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
from .utils import (
    TEMP_SESSION_KEY,
    decode_keycloak_jwt,
    get_redis_key_from_session,
    handle_login_request,
    handle_token_request,
)

blueprint = Blueprint("web_auth", __name__, url_prefix="/auth")

KC_SUFFIX = "kc_oidc_client"
SCOPE = ["profile", "email", "openid"]


def get_valid_token(headers):
    """Look for a fresh and valid token in the session."""
    if headers.get("X-Requested-With") == "XMLHttpRequest" and "sub" in session:
        redis_key = get_redis_key_from_session(key_suffix=KC_SUFFIX)

        keycloak_oidc_client = current_app.store.get_oauth_client(redis_key)
        if hasattr(keycloak_oidc_client, "access_token"):
            return keycloak_oidc_client.access_token

        current_app.logger.warning(
            "The user does not have backend access tokens.",
            exc_info=True,
        )

    return None


LOGIN_SEQUENCE = ("web_auth.login", "gitlab_auth.login")


@blueprint.route("/login/next")
def login_next():
    session["login_seq"] += 1
    if session["login_seq"] < len(LOGIN_SEQUENCE):
        next_login = LOGIN_SEQUENCE[session["login_seq"]]
        return render_template(
            "redirect.html",
            redirect_url=urljoin(current_app.config["HOST_NAME"], url_for(next_login)),
        )
    else:
        return redirect(session["ui_redirect_url"])


@blueprint.route("/login")
def login():
    """Log in with Keycloak."""
    session.clear()
    session["ui_redirect_url"] = request.args.get("redirect_url")
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

    return response


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
        # NOTE: Do not delete GL client because CLI login uses it for authentication
        # current_app.store.delete(get_redis_key_from_session(key_suffix=GL_SUFFIX))
        current_app.store.delete(get_redis_key_from_session(key_suffix=KC_SUFFIX))
    session.clear()

    logout_pages = []
    if current_app.config["LOGOUT_GITLAB_UPON_RENKU_LOGOUT"]:
        logout_pages = [
            urljoin(current_app.config["HOST_NAME"], url_for("gitlab_auth.logout"))
        ]
    logout_pages.append(
        f"{current_app.config['OIDC_ISSUER']}/protocol/openid-connect/logout"
    )

    return render_template(
        "redirect_logout.html",
        redirect_url=request.args.get("redirect_url"),
        logout_pages=logout_pages,
        len=len(logout_pages),
    )
