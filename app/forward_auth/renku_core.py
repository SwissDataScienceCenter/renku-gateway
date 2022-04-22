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
"""Add the headers for the Renku core service."""
from base64 import b64encode

from flask import current_app

from ..common.utils import GL_SUFFIX, KC_SUFFIX, build_redis_key, decode_keycloak_jwt


def add_auth_headers(sub, headers):
    keycloak_oidc_client = current_app.store.get_oauth_client(
        build_redis_key(sub, key_suffix=KC_SUFFIX)
    )
    headers["Renku-User"] = keycloak_oidc_client.token["id_token"]

    gitlab_oauth_client = current_app.store.get_oauth_client(
        build_redis_key(sub, key_suffix=GL_SUFFIX)
    )
    headers["Authorization"] = "Bearer {}".format(gitlab_oauth_client.access_token)

    # TODO: The subsequent information is now included in the JWT sent in the
    # TODO: "Renku-User" header. It can be removed once the core-service
    # TODO: relies on the new header.
    access_token_dict = decode_keycloak_jwt(keycloak_oidc_client.access_token)
    headers["Renku-user-id"] = access_token_dict["sub"]
    headers["Renku-user-email"] = b64encode(access_token_dict["email"].encode())
    headers["Renku-user-fullname"] = b64encode(access_token_dict["name"].encode())

    return headers
