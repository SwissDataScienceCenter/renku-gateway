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

import json
import re
import time
import urllib.parse
from urllib.parse import quote_plus, urljoin

import jwt
from oic import rndstr
from oic.oauth2.grant import Token
from oic.oic import Client
from oic.oic.message import AuthorizationResponse, RegistrationResponse
from oic.utils.authn.client import CLIENT_AUTHN_METHOD
from quart import (
    Blueprint,
    Response,
    current_app,
    redirect,
    render_template,
    request,
    session,
    url_for,
)


blueprint = Blueprint("web_auth", __name__, url_prefix="/auth")

# Note that this part of the service should be seen as the server-side part of the UI or

JWT_ALGORITHM = "RS256"
SCOPE = ["openid"]


@blueprint.before_request
def before_first():
    if current_app.config.get("KEYCLOAK_OIC_CLIENT"):
        return

    """Fake the response from registering the client through the API."""
    try:
        # We prepare the OIC client instance with the necessary configurations.
        keycloak_oic_client = Client(client_authn_method=CLIENT_AUTHN_METHOD)
        keycloak_oic_client.provider_config(issuer=current_app.config["OIDC_ISSUER"])
    except:
        pass

    client_reg = RegistrationResponse(
        client_id=current_app.config["OIDC_CLIENT_ID"],
        client_secret=current_app.config["OIDC_CLIENT_SECRET"],
    )
    keycloak_oic_client.store_registration_info(client_reg)
    current_app.config["KEYCLOAK_OIC_CLIENT"] = keycloak_oic_client


def get_valid_token(headers):
    """
    Look for a fresh and valid token, first in headers, then in the session.

    If a refresh token is available, it can be swapped for an access token.
    """
    from .. import store

    keycloak_oic_client = current_app.config.get("KEYCLOAK_OIC_CLIENT")

    m = re.search(
        r"bearer (?P<token>.+)", headers.get("Authorization", ""), re.IGNORECASE
    )

    if m:
        if jwt.decode(m.group("token"), verify=False).get("typ") in [
            "Offline",
            "Refresh",
        ]:
            current_app.logger.debug("Swapping the token")
            to = Token(resp={"refresh_token": m.group("token")})
            token_response = keycloak_oic_client.do_access_token_refresh(token=to)

            if "access_token" in token_response:
                try:
                    a = jwt.decode(
                        token_response["access_token"],
                        current_app.config["OIDC_PUBLIC_KEY"],
                        algorithms=JWT_ALGORITHM,
                        audience=current_app.config["OIDC_CLIENT_ID"],
                    )
                    return token_response
                except:
                    return None
        else:
            try:
                jwt.decode(
                    m.group("token"),
                    current_app.config["OIDC_PUBLIC_KEY"],
                    algorithms=JWT_ALGORITHM,
                    audience=current_app.config["OIDC_CLIENT_ID"],
                )

                return {"access_token": m.group("token")}

            except:
                return None
    else:
        if headers.get("X-Requested-With") == "XMLHttpRequest" and "token" in session:
            try:
                jwt.decode(
                    session.get("token"),
                    current_app.config["OIDC_PUBLIC_KEY"],
                    algorithms=JWT_ALGORITHM,
                    audience=current_app.config["OIDC_CLIENT_ID"],
                )
                return {"access_token": session.get("token")}

            except:

                a = jwt.decode(session.get("token"), verify=False)
                refresh_token = store.get(
                    get_key_for_user(a, "kc_refresh_token")
                ).decode()

                current_app.logger.debug("Refreshing the token")
                to = Token(resp={"refresh_token": refresh_token})

                token_response = keycloak_oic_client.do_access_token_refresh(token=to)

                if "access_token" in token_response:
                    try:
                        a = jwt.decode(
                            token_response["access_token"],
                            current_app.config["OIDC_PUBLIC_KEY"],
                            algorithms=JWT_ALGORITHM,
                            audience=current_app.config["OIDC_CLIENT_ID"],
                        )
                        session["token"] = token_response["access_token"]
                        store.put(
                            get_key_for_user(a, "kc_access_token"),
                            token_response["access_token"].encode(),
                        )
                        store.put(
                            get_key_for_user(a, "kc_refresh_token"),
                            token_response["refresh_token"].encode(),
                        )
                        store.put(
                            get_key_for_user(a, "kc_id_token"),
                            json.dumps(token_response["id_token"].to_dict()).encode(),
                        )
                        return token_response
                    except:
                        return None

    return None


def get_key_for_user(token, name):
    """ Create a base-64 encoded key for the redis store """
    from base64 import b64encode

    key = "cache_{}_{}".format(token.get("sub"), name)
    return b64encode(key.encode()).decode("utf-8")


LOGIN_SEQUENCE = ["gitlab_auth.login", "jupyterhub_auth.login"]


@blueprint.route("/login/next")
async def login_next():

    if session["login_seq"] < len(LOGIN_SEQUENCE):
        return await render_template(
            "redirect.html", redirect_url=url_for(LOGIN_SEQUENCE[session["login_seq"]])
        )
    else:
        return redirect(session["ui_redirect_url"])


@blueprint.route("/login")
async def login():

    keycloak_oic_client = current_app.config.get("KEYCLOAK_OIC_CLIENT")

    state = rndstr()

    session["state"] = state
    session["login_seq"] = 0
    session["ui_redirect_url"] = request.args.get("redirect_url")
    session["cli_token"] = request.args.get("cli_token")
    if session["cli_token"]:
        session["ui_redirect_url"] = current_app.config["HOST_NAME"] + url_for(
            "web_auth.info"
        )

    args = {
        "client_id": current_app.config["OIDC_CLIENT_ID"],
        "response_type": "code",
        "scope": SCOPE,
        "redirect_uri": current_app.config["HOST_NAME"] + url_for("web_auth.token"),
        "state": state,
    }
    auth_req = keycloak_oic_client.construct_AuthorizationRequest(request_args=args)
    login_url = auth_req.request(keycloak_oic_client.authorization_endpoint)
    response = await current_app.make_response(redirect(login_url))

    return response


@blueprint.route("/token")
async def token():
    from .. import store

    keycloak_oic_client = current_app.config.get("KEYCLOAK_OIC_CLIENT")

    # This is more about parsing the request data than any response data....
    authorization_parameters = keycloak_oic_client.parse_response(
        AuthorizationResponse,
        info=request.query_string.decode("utf-8"),
        sformat="urlencoded",
    )

    if session.get("state") != authorization_parameters["state"]:
        return "Something went wrong while trying to log you in."

    token_response = keycloak_oic_client.do_access_token_request(
        scope=SCOPE,
        state=authorization_parameters["state"],
        request_args={
            "code": authorization_parameters["code"],
            "redirect_uri": current_app.config["HOST_NAME"] + url_for("web_auth.token"),
        },
    )

    # chain logins
    response = await current_app.make_response(redirect(url_for("web_auth.login_next")))

    a = jwt.decode(
        token_response["access_token"],
        current_app.config["OIDC_PUBLIC_KEY"],
        algorithms=JWT_ALGORITHM,
        audience=current_app.config["OIDC_CLIENT_ID"],
    )
    session["token"] = token_response["access_token"]
    store.put(
        get_key_for_user(a, "kc_access_token"), token_response["access_token"].encode()
    )
    store.put(
        get_key_for_user(a, "kc_refresh_token"),
        token_response["refresh_token"].encode(),
    )
    store.put(
        get_key_for_user(a, "kc_id_token"),
        json.dumps(token_response["id_token"].to_dict()).encode(),
    )

    # we can already tell the CLI which token to use
    if session.get("cli_token"):
        current_app.logger.debug(
            "Notification for request {}".format(session.get("cli_token"))
        )

        key = "cli_{}".format(
            hashlib.sha256(session.get("cli_token").encode()).hexdigest()
        )
        store.put(
            key,
            json.dumps(
                {
                    "access_token": token_response["access_token"],
                    "refresh_token": token_response["refresh_token"],
                }
            ).encode(),
        )

    return response


@blueprint.route("/info")
async def info():
    from .. import store

    t = request.args.get("cli_token")
    if t:
        timeout = 120
        key = "cli_{}".format(hashlib.sha256(t.encode()).hexdigest())
        current_app.logger.debug(
            "Waiting for Keycloak callback for request {}".format(t)
        )
        val = store.get(key)
        while not val and timeout > 0:
            time.sleep(3)
            timeout -= 3
            val = store.get(key)
        if val:
            store.delete(key)
            return val
        else:
            current_app.logger.debug("Timeout while waiting for request {}".format(t))
            return '{"error": "timeout"}'
    else:

        if "token" not in session:
            return await current_app.make_response(
                redirect(
                    "{}?redirect_url={}".format(
                        url_for("web_auth.login"), quote_plus(url_for("web_auth.info"))
                    )
                )
            )

        try:
            a = jwt.decode(
                session["token"],
                current_app.config["OIDC_PUBLIC_KEY"],
                algorithms=JWT_ALGORITHM,
                audience=current_app.config["OIDC_CLIENT_ID"],
            )  # TODO: logout and redirect if fails because of expired

            return (
                "You can copy/paste the following tokens if needed "
                "and close this page: "
                "<br> Access Token: {}<br>Refresh Token: {}".format(
                    store.get(get_key_for_user(a, "kc_access_token")).decode(),
                    store.get(get_key_for_user(a, "kc_refresh_token")).decode(),
                )
            )

        except jwt.ExpiredSignatureError:
            return await current_app.make_response(
                redirect(
                    "{}?redirect_url={}".format(
                        url_for("web_auth.login"), quote_plus(url_for("web_auth.info"))
                    )
                )
            )


@blueprint.route("/user")
async def user():
    from .. import store

    if "token" not in session:
        return await current_app.make_response(
            redirect(
                "{}?redirect_url={}".format(
                    url_for("web_auth.login"), quote_plus(url_for("web_auth.user"))
                )
            )
        )
    try:
        a = jwt.decode(
            session["token"],
            current_app.config["OIDC_PUBLIC_KEY"],
            algorithms=JWT_ALGORITHM,
            audience=current_app.config["OIDC_CLIENT_ID"],
        )  # TODO: logout and redirect if fails because of expired

        return store.get(get_key_for_user(a, "kc_id_token")).decode()

    except jwt.ExpiredSignatureError:
        return await current_app.make_response(
            redirect(
                "{}?redirect_url={}".format(
                    url_for("web_auth.login"), quote_plus(url_for("web_auth.user"))
                )
            )
        )


@blueprint.route("/user-profile")
async def user_profile():
    return await current_app.make_response(
        redirect("{}/account?referrer=renku".format(current_app.config["OIDC_ISSUER"]))
    )


@blueprint.route("/logout")
async def logout():
    from .. import store

    logout_url = "{}/protocol/openid-connect/logout?{}".format(
        current_app.config["OIDC_ISSUER"],
        urllib.parse.urlencode(
            {
                "redirect_uri": current_app.config["HOST_NAME"]
                + url_for("gitlab_auth.logout")
            }
        ),
    )

    if request.args.get("gitlab_logout"):
        if "logout_from" in session:
            session.clear()
            return await render_template(
                "redirect_logout.html",
                redirect_url="/",
                logout_page=url_for("jupyterhub_auth.logout"),
            )
        else:
            return await current_app.make_response(
                redirect(current_app.config["GITLAB_URL"])
            )

    if "token" in session:
        a = jwt.decode(session["token"], verify=False)

        # cleanup the session in redis immediately
        cookie_val = request.cookies.get("session").split(".")[0]
        store.delete(cookie_val)
        session.clear()

        for k in store.keys(prefix=get_key_for_user(a, "")):
            store.delete(k)

        session["logout_from"] = "Renku"

    return await current_app.make_response(redirect(logout_url))
