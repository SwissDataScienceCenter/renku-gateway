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

from urllib.parse import urljoin, urlencode

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
    get_redis_key_from_session,
    handle_login_request,
    TEMP_SESSION_KEY,
    handle_token_request,
)

blueprint = Blueprint("web_auth", __name__, url_prefix="/auth")

KC_SUFFIX = "kc_oidc_client"
SCOPES = ["openid"]


def get_valid_token(headers):
    """
    Look for a fresh and valid token, first in headers, then in the session.

    If a refresh token is available, it can be swapped for an access token.
    """

    # TODO: uncomment (and fix) when enabling CLI login.
    # m = re.search(
    #     r"bearer (?P<token>.+)", headers.get("Authorization", ""), re.IGNORECASE
    # )

    # if m:
    #     if jwt.decode(m.group("token"), verify=False).get("typ") in [
    #         "Offline",
    #         "Refresh",
    #     ]:
    #         current_app.logger.debug("Swapping the token")
    #         to = Token(resp={"refresh_token": m.group("token")})

    #         if "access_token" in token_response:
    #             try:
    #                 a = jwt.decode(
    #                     token_response["access_token"],
    #                     current_app.config["OIDC_PUBLIC_KEY"],
    #                     algorithms=JWT_ALGORITHM,
    #                     audience=current_app.config["OIDC_CLIENT_ID"],
    #                 )
    #                 return token_response
    #             except:
    #                 return None
    #     else:
    #         try:
    #             jwt.decode(
    #                 m.group("token"),
    #                 current_app.config["OIDC_PUBLIC_KEY"],
    #                 algorithms=JWT_ALGORITHM,
    #                 audience=current_app.config["OIDC_CLIENT_ID"],
    #             )

    #             return {"access_token": m.group("token")}

    #         except:
    #             return None
    # else:
    if headers.get("X-Requested-With") == "XMLHttpRequest" and "sub" in session:
        redis_key = get_redis_key_from_session(key_suffix=KC_SUFFIX)
        keycloak_oidc_client = current_app.store.get_oauth_client(redis_key)
        return {"access_token": keycloak_oidc_client.access_token}

    return None


LOGIN_SEQUENCE = ["web_auth.login", "gitlab_auth.login", "jupyterhub_auth.login"]


@blueprint.route("/login/next")
def login_next():
    session["login_current_seq"] += 1
    if session["login_current_seq"] < len(session["login_sequence"]):
        return render_template(
            "redirect.html",
            redirect_url=urljoin(
                current_app.config["HOST_NAME"],
                url_for(session["login_sequence"][session["login_current_seq"]]),
            ),
        )
    else:
        return redirect(session["ui_redirect_url"])


@blueprint.route("/login")
def login():
    """Log in with Keycloak."""
    session.clear()
    session["ui_redirect_url"] = request.args.get("redirect_url")
    session["cli_token"] = request.args.get("cli_token")
    session["login_current_seq"] = 0

    # verify if the login should target a single service
    target = request.args.get("target")
    if target:
        target_login = next(seq for seq in LOGIN_SEQUENCE[1:] if seq.startswith(target))
        if not target_login:
            raise ValueError(f"The target service is not available for login: {target}")
        session["login_sequence"] = [LOGIN_SEQUENCE[0], target_login]
    else:
        session["login_sequence"] = LOGIN_SEQUENCE

    # if a token refresh is required, first do it
    refresh = request.args.get("refresh")
    if refresh:
        session["refresh"] = True

    provider_app = KeycloakProviderApp(
        client_id=current_app.config["OIDC_CLIENT_ID"],
        client_secret=current_app.config["OIDC_CLIENT_SECRET"],
        base_url=current_app.config["OIDC_ISSUER"],
    )
    return handle_login_request(
        provider_app,
        urljoin(current_app.config["HOST_NAME"], url_for("web_auth.token")),
        KC_SUFFIX,
        SCOPES,
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

    # cleanup services tokens when refreshing
    refresh = session.pop("refresh", None)
    if refresh:
        suffixes = ["GL_SUFFIX", "JH_SUFFIX"]
        for suffix in suffixes:
            target = get_redis_key_from_session(key_suffix=suffix)
            if target:
                current_app.store.delete(target)

    return response


# TODO: Uncomment and fix when implementing CLI login
# @blueprint.route("/info")
# async def info():
#     from .. import store

#     t = request.args.get("cli_token")
#     if t:
#         timeout = 120
#         key = "cli_{}".format(hashlib.sha256(t.encode()).hexdigest())
#         current_app.logger.debug(
#             "Waiting for Keycloak callback for request {}".format(t)
#         )
#         val = current_app.store.get(key)
#         while not val and timeout > 0:
#             time.sleep(3)
#             timeout -= 3
#             val = current_app.store.get(key)
#         if val:
#             current_app.store.delete(key)
#             return val
#         else:
#             current_app.logger.debug("Timeout while waiting for request {}".format(t))
#             return '{"error": "timeout"}'
#     else:

#         if "token" not in session:
#             return await current_app.make_response(
#                 redirect(
#                     "{}?redirect_url={}".format(
# noqa                         url_for("web_auth.login"), quote_plus(url_for("web_auth.info"))
#                     )
#                 )
#             )

#         try:
#             a = jwt.decode(
#                 session["token"],
#                 current_app.config["OIDC_PUBLIC_KEY"],
#                 algorithms=JWT_ALGORITHM,
#                 audience=current_app.config["OIDC_CLIENT_ID"],
#             )  # TODO: logout and redirect if fails because of expired

#             return (
#                 "You can copy/paste the following tokens if needed "
#                 "and close this page: "
#                 "<br> Access Token: {}<br>Refresh Token: {}".format(
# noqa                    current_app.store.get(get_redis_key(a, "kc_access_token")).decode(),
# noqa                    current_app.store.get(get_redis_key(a, "kc_refresh_token")).decode(),
#                 )
#             )

#         except jwt.ExpiredSignatureError:
#             return await current_app.make_response(
#                 redirect(
#                     "{}?redirect_url={}".format(
# noqa                        url_for("web_auth.login"), quote_plus(url_for("web_auth.info"))
#                     )
#                 )
#             )


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

    logout_url = "{}/protocol/openid-connect/logout?{}".format(
        current_app.config["OIDC_ISSUER"],
        urlencode(
            {
                "redirect_uri": urljoin(
                    current_app.config["HOST_NAME"], url_for("gitlab_auth.logout")
                )
            }
        ),
    )

    if request.args.get("gitlab_logout"):
        if "logout_from" in session:
            session.clear()
            return render_template(
                "redirect_logout.html",
                redirect_url="/",
                logout_page=urljoin(
                    current_app.config["HOST_NAME"], url_for("jupyterhub_auth.logout")
                ),
            )
        else:
            return current_app.make_response(redirect(current_app.config["GITLAB_URL"]))

    if "sub" in session:
        current_app.store.delete(get_redis_key_from_session(key_suffix=GL_SUFFIX))
        current_app.store.delete(get_redis_key_from_session(key_suffix=JH_SUFFIX))
        current_app.store.delete(get_redis_key_from_session(key_suffix=KC_SUFFIX))

    session.clear()
    session["logout_from"] = "Renku"

    return current_app.make_response(redirect(logout_url))
