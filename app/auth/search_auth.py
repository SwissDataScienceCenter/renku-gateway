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
"""Add the headers for the Renku searchservice."""
import re

from ..config import KC_SUFFIX
from .utils import get_or_set_keycloak_client, get_redis_key_from_token


class SearchHeaders:
    def process(self, request, headers):
        m = re.search(
            r"bearer (?P<token>.+)", headers.get("Authorization", ""), re.IGNORECASE
        )
        if m:
            access_token = m.group("token")

            keycloak_redis_key = get_redis_key_from_token(
                access_token, key_suffix=KC_SUFFIX
            )
            keycloak_oidc_client = get_or_set_keycloak_client(keycloak_redis_key)

            headers["Renku-Auth-Id-Token"] = keycloak_oidc_client.token["id_token"]
        else:
            headers["Renku-Auth-Anon-Id"] = request.cookies.get("anon-id", "")

        return headers
