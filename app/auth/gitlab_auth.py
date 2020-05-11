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

import json
import re
import urllib
from urllib.parse import urljoin

import jwt
from oic import rndstr
from oic.oauth2.grant import Token
from oic.oic import Client
from oic.oic.message import AuthorizationResponse, RegistrationResponse
from oic.utils.authn.client import CLIENT_AUTHN_METHOD
from oic.utils.keyio import KeyJar
from quart import Blueprint, Response, current_app, redirect, request, session, url_for

from .web import JWT_ALGORITHM, get_key_for_user

blueprint = Blueprint("gitlab_auth", __name__, url_prefix="/auth/gitlab")


class GitlabUserToken:
    def __init__(self, header_field="Authorization", header_prefix="Bearer "):
        self.header_field = header_field
        self.header_prefix = header_prefix

    def process(self, request, headers):
        from .. import store

        m = re.search(
            r"bearer (?P<token>.+)", headers.get("Authorization", ""), re.IGNORECASE
        )
        if m:
            # current_app.logger.debug('outgoing headers: {}'.format(json.dumps(headers))
            access_token = m.group("token")
            decodentoken = jwt.decode(
                access_token,
                current_app.config["OIDC_PUBLIC_KEY"],
                algorithms=JWT_ALGORITHM,
                audience=current_app.config["OIDC_CLIENT_ID"],
            )

            gl_token = store.get(get_key_for_user(decodentoken, "gl_access_token"))
            headers[self.header_field] = "{}{}".format(
                self.header_prefix, gl_token.decode()
            )
            headers[
                "Renku-Token"
            ] = access_token  # can be needed later in the request processing

        else:
            # current_app.logger.debug("No authorization header, returning empty auth headers")
            pass

        return headers


SCOPE = ["openid", "api", "read_user", "read_repository"]


@blueprint.before_request
def create_gitlab_oic_client():
    if current_app.config.get("GITLAB_OIC_CLIENT"):
        return

    try:
        # We prepare the OIC client instance with the necessary configurations.
        gitlab_oic_client = Client(client_authn_method=CLIENT_AUTHN_METHOD)
        gitlab_oic_client.provider_config(
            issuer=current_app.config["GITLAB_URL"], keys=False,
        )
    except Exception:
        pass

    # This fakes the response we would get from registering the client through the API
    client_reg = RegistrationResponse(
        client_id=current_app.config["GITLAB_CLIENT_ID"],
        client_secret=current_app.config["GITLAB_CLIENT_SECRET"],
    )
    gitlab_oic_client.store_registration_info(client_reg)

    # gitlab /.well-known/openid-configuration doesn't take into account
    # the protocol for generating its URLs
    # so we have to manualy fix them here
    gitlab_oic_client.authorization_endpoint = "{}/oauth/authorize".format(
        current_app.config["GITLAB_URL"]
    )
    gitlab_oic_client.token_endpoint = "{}/oauth/token".format(
        current_app.config["GITLAB_URL"]
    )
    gitlab_oic_client.userinfo_endpoint = "{}/oauth/userinfo".format(
        current_app.config["GITLAB_URL"]
    )
    gitlab_oic_client.jwks_uri = "{}/oauth/discovery/keys".format(
        current_app.config["GITLAB_URL"]
    )
    gitlab_oic_client.keyjar = KeyJar()
    gitlab_oic_client.keyjar.load_keys(
        {
            "jwks_uri": "{0}/oauth/discovery/keys".format(
                current_app.config["GITLAB_URL"]
            )
        },
        current_app.config["GITLAB_URL"],
    )

    current_app.config["GITLAB_OIC_CLIENT"] = gitlab_oic_client


@blueprint.route("/login")
def login():
    """Login with GitLab."""

    gitlab_oic_client = current_app.config.get("GITLAB_OIC_CLIENT")

    state = rndstr()

    session["login_seq"] += 1
    session["gitlab_state"] = state

    args = {
        "client_id": current_app.config["GITLAB_CLIENT_ID"],
        "response_type": "code",
        "scope": SCOPE,
        "redirect_uri": current_app.config["HOST_NAME"] + url_for("gitlab_auth.token"),
        "state": state,
    }
    auth_req = gitlab_oic_client.construct_AuthorizationRequest(request_args=args)
    login_url = auth_req.request(gitlab_oic_client.authorization_endpoint)
    response = current_app.make_response(redirect(login_url))
    return response


@blueprint.route("/token")
async def token():
    from .. import store

    gitlab_oic_client = current_app.config.get("GITLAB_OIC_CLIENT")

    # This is more about parsing the request data than any response data....
    authorization_parameters = gitlab_oic_client.parse_response(
        AuthorizationResponse,
        info=request.query_string.decode("utf-8"),
        sformat="urlencoded",
    )

    if session["gitlab_state"] != authorization_parameters["state"]:
        return "Something went wrong while trying to log you in."

    token_response = gitlab_oic_client.do_access_token_request(
        scope=SCOPE,
        state=authorization_parameters["state"],
        request_args={
            "code": authorization_parameters["code"],
            "redirect_uri": current_app.config["HOST_NAME"]
            + url_for("gitlab_auth.token"),
        },
    )

    a = jwt.decode(session["token"], verify=False)
    store.put(
        get_key_for_user(a, "gl_access_token"), token_response["access_token"].encode()
    )
    store.put(
        get_key_for_user(a, "gl_refresh_token"),
        token_response["refresh_token"].encode(),
    )
    store.put(
        get_key_for_user(a, "gl_id_token"),
        json.dumps(token_response["id_token"].to_dict()).encode(),
    )

    response = await current_app.make_response(redirect(url_for("web_auth.login_next")))

    return response


def get_gitlab_refresh_token(access_token):
    from .. import store

    gitlab_oic_client = current_app.config.get("GITLAB_OIC_CLIENT")

    access_token = jwt.decode(
        access_token,
        current_app.config["OIDC_PUBLIC_KEY"],
        algorithms=JWT_ALGORITHM,
        audience=current_app.config["OIDC_CLIENT_ID"],
    )
    to = Token(
        resp={
            "refresh_token": store.get(
                get_key_for_user(access_token, "gl_refresh_token")
            )
        }
    )
    refresh_token_response = gitlab_oic_client.do_access_token_refresh(token=to)
    if "access_token" in refresh_token_response:
        store.put(
            get_key_for_user(access_token, "gl_access_token"),
            refresh_token_response["access_token"].encode(),
        )
        store.put(
            get_key_for_user(access_token, "gl_refresh_token"),
            refresh_token_response["refresh_token"].encode(),
        )
    return refresh_token_response.get("access_token")


@blueprint.route("/logout")
def logout():
    logout_url = current_app.config["GITLAB_URL"] + "/users/sign_out"
    response = current_app.make_response(redirect(logout_url))
    return response
