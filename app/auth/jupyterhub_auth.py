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

import json
import re
from urllib.parse import parse_qs, urlencode, urljoin

import jwt
import requests
from oic import rndstr
from quart import Blueprint, current_app, redirect, request, Response, session, url_for

from .web import JWT_ALGORITHM, get_key_for_user


blueprint = Blueprint("jupyterhub_auth", __name__, url_prefix="/auth/jupyterhub")


class JupyterhubUserToken:
    def process(self, request, headers):
        from .. import store

        m = re.search(
            r"bearer (?P<token>.+)", headers.get("Authorization", ""), re.IGNORECASE
        )
        if m:
            # current_app.logger.debug('Authorization header present, token exchange')
            access_token = m.group("token")
            decodentoken = jwt.decode(
                access_token,
                current_app.config["OIDC_PUBLIC_KEY"],
                algorithms=JWT_ALGORITHM,
                audience=current_app.config["OIDC_CLIENT_ID"],
            )

            jh_token = store.get(get_key_for_user(decodentoken, "jh_access_token"))
            headers["Authorization"] = "token {}".format(jh_token.decode())

            # current_app.logger.debug('outgoing headers: {}'.format(json.dumps(headers)))
        else:
            # current_app.logger.debug("No authorization header, returning empty auth headers")
            headers.pop("Authorization", None)

        return headers


JUPYTERHUB_OAUTH2_PATH = "/hub/api/oauth2"


@blueprint.route("/login")
def login():
    state = rndstr()

    session["login_seq"] += 1
    session["jupyterhub_state"] = state

    args = {
        "client_id": current_app.config["JUPYTERHUB_CLIENT_ID"],
        "response_type": "code",
        "redirect_uri": current_app.config["HOST_NAME"]
        + url_for("jupyterhub_auth.token"),
        "state": state,
    }
    url = current_app.config["JUPYTERHUB_URL"] + JUPYTERHUB_OAUTH2_PATH + "/authorize"
    login_url = "{}?{}".format(url, urlencode(args))
    response = current_app.make_response(redirect(login_url))
    return response


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
async def token():
    from .. import store

    authorization_parameters = parse_qs(request.query_string.decode())

    if session["jupyterhub_state"] != authorization_parameters["state"][0]:
        return "Something went wrong while trying to log you in."

    token_response = requests.post(
        current_app.config["JUPYTERHUB_URL"] + JUPYTERHUB_OAUTH2_PATH + "/token",
        data={
            "client_id": current_app.config["JUPYTERHUB_CLIENT_ID"],
            "client_secret": current_app.config["JUPYTERHUB_CLIENT_SECRET"],
            "state": session["jupyterhub_state"],
            "code": authorization_parameters["code"][0],
            "grant_type": "authorization_code",
            "redirect_uri": current_app.config["HOST_NAME"]
            + url_for("jupyterhub_auth.token"),
        },
    )

    a = jwt.decode(session["token"], verify=False)
    store.put(
        get_key_for_user(a, "jh_access_token"),
        token_response.json().get("access_token").encode(),
    )

    response = await current_app.make_response(redirect(url_for("web_auth.login_next")))

    return response


@blueprint.route("/logout")
def logout():
    logout_url = current_app.config["JUPYTERHUB_URL"] + "/hub/logout"
    response = current_app.make_response(redirect(logout_url))

    return response
