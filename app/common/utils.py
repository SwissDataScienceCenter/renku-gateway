# -*- coding: utf-8 -*-
#
# Copyright 2018-2020 - Swiss Data Science Center (SDSC)
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

from datetime import datetime, timedelta
from urllib.parse import urljoin

import jwt
from flask import current_app, redirect, session, url_for
import requests

from .oauth_client import RenkuWebApplicationClient
from .oauth_provider_app import PROVIDER_KINDS

JWT_ALGORITHM = "RS256"
KC_SUFFIX = "kc_oidc_client"
GL_SUFFIX = "gl_oauth_client"


def decode_keycloak_jwt(token):
    """Decode a keycloak access token (JWT) and check the signature."""
    try:
        return jwt.decode(
            token,
            current_app.config["OIDC_PUBLIC_KEY"],
            algorithms=JWT_ALGORITHM,
            audience=current_app.config["OIDC_CLIENT_ID"],
        )
    except jwt.exceptions.InvalidSignatureError as err:
        if datetime.now() - current_app.config["PUBLIC_KEY_DATETIME"] > timedelta(
            minutes=1
        ):
            print(datetime.now() - current_app.config["PUBLIC_KEY_DATETIME"])
            current_app.logger.info("Refetching public key...")
            load_public_key(max_age=timedelta(minutes=1))
            decode_keycloak_jwt(token)
        else:
            raise err


def build_redis_key(sub_claim, key_suffix=""):
    return "cache_{}_{}".format(sub_claim, key_suffix)


def get_redis_key_from_token(token, key_suffix=""):
    """Get the redis store from a keycloak access_token."""
    decoded_token = decode_keycloak_jwt(token)
    return build_redis_key(decoded_token["sub"], key_suffix=key_suffix)


def handle_login_request(provider_app, redirect_path, key_suffix, scope):
    """Logic to handle the login requests, avoids duplication"""
    oauth_client = RenkuWebApplicationClient(
        provider_app=provider_app,
        redirect_url=urljoin(current_app.config["HOST_NAME"], redirect_path),
        scope=scope,
        max_lifetime=None,
    )
    authorization_url = oauth_client.get_authorization_url()
    redis_key = build_redis_key(session["id"], key_suffix=key_suffix)
    current_app.store.set_oauth_client(redis_key, oauth_client)
    return current_app.make_response(redirect(authorization_url))


def handle_token_request(request, key_suffix):
    """Logic to handle the token requests, avoids duplication"""
    redis_key = build_redis_key(session["id"], key_suffix=key_suffix)
    oauth_client = current_app.store.get_oauth_client(redis_key)
    current_app.store.delete(redis_key)
    oauth_client.fetch_token(request.url)
    if oauth_client.provider_app.kind == PROVIDER_KINDS["KEYCLOAK"]:
        session["sub"] = decode_keycloak_jwt(oauth_client.access_token)["sub"]
    redis_key = build_redis_key(session["sub"], key_suffix=key_suffix)
    current_app.store.set_oauth_client(redis_key, oauth_client)
    response = current_app.make_response(
        redirect(
            urljoin(current_app.config["HOST_NAME"], url_for("web_auth.login_next"))
        )
    )
    return response


def load_public_key(max_age=timedelta(0)):
    fetch_new_key = False

    if "OIDC_PUBLIC_KEY" not in current_app.config:
        fetch_new_key = True
    elif datetime.now() - current_app.config["PUBLIC_KEY_DATETIME"] > max_age:
        fetch_new_key = True

    if fetch_new_key:
        current_app.logger.info(
            "Obtaining Keycloak config from {}".format(
                current_app.config["KEYCLOAK_WELL_KNOWN_URL"]
            )
        )
        current_app.config["KEYCLOAK_WELL_KNOWN_CONFIG"] = requests.get(
            current_app.config["KEYCLOAK_WELL_KNOWN_URL"]
        ).json()

        current_app.logger.info(
            "Obtaining public key from {}".format(
                current_app.config["KEYCLOAK_WELL_KNOWN_CONFIG"]["issuer"]
            )
        )

        raw_key = requests.get(
            current_app.config["KEYCLOAK_WELL_KNOWN_CONFIG"]["issuer"]
        ).json()["public_key"]
        current_app.config[
            "OIDC_PUBLIC_KEY"
        ] = "-----BEGIN PUBLIC KEY-----\n{}\n-----END PUBLIC KEY-----".format(raw_key)
        current_app.config["PUBLIC_KEY_DATETIME"] = datetime.now()
        current_app.logger.info(current_app.config["OIDC_PUBLIC_KEY"])
