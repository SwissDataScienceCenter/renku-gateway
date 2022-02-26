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
"""Implement Keycloak authentication workflow for CLI."""

import base64

from flask import current_app

from .gitlab_auth import GL_SUFFIX
from .utils import get_redis_key_from_token

SCOPE = ["profile", "email", "openid"]


class RenkuCLIGitlabAuthHeaders:
    @staticmethod
    def process(request, headers):
        if not request.authorization:
            return headers

        access_token = request.authorization.password
        if access_token:
            redis_key = get_redis_key_from_token(access_token, key_suffix=GL_SUFFIX)
            gitlab_oauth_client = current_app.store.get_oauth_client(redis_key)
            if gitlab_oauth_client:
                gitlab_access_token = gitlab_oauth_client.access_token
                user_pass = f"oauth2:{gitlab_access_token}".encode("utf-8")
                basic_auth = base64.b64encode(user_pass).decode("ascii")
                headers["Authorization"] = f"Basic {basic_auth}"

        return headers
