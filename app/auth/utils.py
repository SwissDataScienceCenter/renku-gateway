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
import random
import string
from urllib.parse import urljoin

import jwt
from flask import current_app, redirect, session, url_for

from .oauth_client import RenkuWebApplicationClient

JWT_ALGORITHM = "RS256"
TEMP_SESSION_KEY = "temp_cache_key"


def decode_keycloak_jwt(token):
    """Decode a keycloak access token (JWT) and check the signature"""
    return jwt.decode(
        token,
        current_app.config["OIDC_PUBLIC_KEY"],
        algorithms=JWT_ALGORITHM,
        audience=current_app.config["OIDC_CLIENT_ID"],
    )


def _get_redis_key(sub_claim, key_suffix=""):
    return "cache_{}_{}".format(sub_claim, key_suffix)


def get_redis_key_from_session(key_suffix=""):
    """Create a key for the redis store.
    - use 'sub' claim if already present in session
    - otherwise use temporary cache key if already present in session
    - otherwise use newly created random string and store it
    Note that the session is passed through the app context."""

    if session.get("sub", None):
        return _get_redis_key(session["sub"], key_suffix=key_suffix)

    if session.get(TEMP_SESSION_KEY, None):
        return session[TEMP_SESSION_KEY]

    random_key = "".join(random.choice(string.ascii_lowercase) for i in range(48))
    session[TEMP_SESSION_KEY] = random_key
    return random_key


def get_redis_key_from_token(token, key_suffix=""):
    """Get the redis store from a keycloak access_token."""
    decoded_token = decode_keycloak_jwt(token)
    return _get_redis_key(decoded_token["sub"], key_suffix=key_suffix)


def handle_login_request(provider_app, redirect_path, key_suffix, scope):
    """Logic to handle the login requests, avoids duplication"""
    oauth_client = RenkuWebApplicationClient(
        provider_app=provider_app,
        redirect_url=urljoin(current_app.config["HOST_NAME"], redirect_path),
        scope=scope,
        max_lifetime=current_app.config["MAX_ACCESS_TOKEN_LIFETIME"],
    )
    authorization_url = oauth_client.get_authorization_url()
    redis_key = get_redis_key_from_session(key_suffix=key_suffix)
    current_app.store.set_oauth_client(redis_key, oauth_client)
    return current_app.make_response(redirect(authorization_url))


def handle_token_request(request, key_suffix):
    """Logic to handle the token requests, avoids duplication"""
    redis_key = get_redis_key_from_session(key_suffix=key_suffix)
    oauth_client = current_app.store.get_oauth_client(redis_key, no_refresh=True)
    oauth_client.fetch_token(request.url)
    current_app.store.set_oauth_client(redis_key, oauth_client)
    response = current_app.make_response(
        redirect(
            urljoin(current_app.config["HOST_NAME"], url_for("web_auth.login_next"))
        )
    )
    return response, oauth_client
