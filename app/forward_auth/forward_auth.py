# -*- coding: utf-8 -*-
#
# Copyright 2017-2019 - Swiss Data Science Center (SDSC)
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
import json
import re
import requests
import secrets

from flask import Blueprint, Response, current_app, request

from . import cli, gitlab, renku_core, notebooks
from ..common import utils

blueprint = Blueprint("forward_auth", __name__, url_prefix="/forward-auth")

auth_modules = {
    "gitlab": gitlab,
    "renku": renku_core,
    "notebook": notebooks,
    "cli-gitlab": cli,
}


def get_access_token_from_request(request, auth_case):
    """
    Small helper function which extracts the keycloak access token from the in-
    coming request.
    """

    if auth_case == "cli-gitlab":
        access_token = request.authorization.password
    else:
        try:
            access_token = re.search(
                r"bearer (?P<token>.+)",
                request.headers.get("Authorization", ""),
                re.IGNORECASE,
            ).group("token")
        except AttributeError:
            access_token = None

    return access_token


def validate_access_token(token):
    """
    Validate a keycloak access token (JWT) offline (by verifying the signature)
    or online (by querying keycloak). If valid, return the validated and decoded
    token. If invalid, log some error and return None.

    Note:
    In case offline validation is not explicitly allowed for the client app for
    which the access token was issued (and hence missing in the jwt scope), we
    check the token by querying the userinfo endpoint. This makes it possible
    for tokens to be revoked. Only clients with very short-lived access tokens
    such as a browser-UI should be enabled for offline validation.
    """

    # Note that the token is also validated while decoding
    try:
        decoded_token = utils.decode_keycloak_jwt(token)
    except Exception as err:
        current_app.logger.warning(
            f"{type(err).__name__} raised while decoding incoming access token: {err}",
        )
        return None

    if "offline_validation" in decoded_token["scope"]:
        return decoded_token

    # Query the userinfo endpoint for online validation
    userinfo_response = requests.get(
        current_app.config["KEYCLOAK_WELL_KNOWN_CONFIG"]["userinfo_endpoint"],
        headers={"Authorization": f"Bearer {token}"},
    )
    if userinfo_response.status_code == 200:
        return decoded_token
    else:
        current_app.logger.warning(
            f"Problem validating jwt with Keycloak: {userinfo_response.json()}",
        )
        return None


@blueprint.route("/", methods=["GET"])
def forward_auth():
    """
    This route implements the traefik forward-auth middleware which swaps the
    keycloak access token in the request header for the headers required by the
    respective backend service.
    Note: We follow the example of github and gitlab by letting requests with a
    missing or malformatted authorization header through. However, for invalid
    access tokens we return a 401 even if accessing the respective resource does
    would not need authentication.
    """

    current_app.logger.debug(f"Hitting forward-auth with args: {request.args}")

    auth_case = request.args.get("auth", None)
    if auth_case not in auth_modules:
        current_app.logger.error(
            f"Hitting forward-auth with missing or unknown auth case: {auth_case}"
        )
        return Response("Unknown auth case", status=422)

    # TODO: Remove as soon as UI-server handles anonymous IDs
    if (
        "anon-id" not in request.cookies
        and request.headers.get("X-Requested-With", "") == "XMLHttpRequest"
        and request.headers["X-Forwarded-Uri"] == "/api/user"
        and "Authorization" not in request.headers
    ):
        resp = Response(
            json.dumps({"message": "401 Unauthorized"}),
            content_type="application/json",
            status=401,
        )
        # We make sure the anonymous ID starts with an alphabetic character
        # such that it can be used directly to form k8s resource names.
        resp.set_cookie("anon-id", f"anon-{secrets.token_urlsafe(32)}")
        current_app.logger.debug("Setting anonymous id")
        return resp

    response_headers = {}

    # TODO: This will soon be business of the UI server
    response_headers["Renku-Auth-Anon-Id"] = request.cookies.get("anon-id", "")

    access_token = get_access_token_from_request(request, auth_case)
    if access_token is None:
        return Response(headers=response_headers, status=200)

    decoded_token = validate_access_token(access_token)

    if decoded_token is not None:
        response_headers = auth_modules[auth_case].add_auth_headers(
            decoded_token["sub"], response_headers
        )
        # Note that this Response defines the forwarded headers and NOT
        # a response that will ever go to the client.
        return Response(headers=response_headers, status=200)

    else:
        # Other than the 200 response above, this response will be returned to
        # the client.
        return Response(
            json.dumps({"message": "401 Unauthorized"}),
            status=401,
            mimetype="application/json",
        )
