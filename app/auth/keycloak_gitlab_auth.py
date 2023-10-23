# Copyright - Swiss Data Science Center (SDSC)
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
"""Add the headers for the Renku notebooks service."""
import re

from flask import current_app

from .gitlab_auth import GL_SUFFIX
from .utils import get_redis_key_from_token


class KeycloakGitlabAuthHeaders:
    def process(self, request, headers):
        m = re.search(
            r"bearer (?P<token>.+)", headers.get("Authorization", ""), re.IGNORECASE
        )
        if m:
            access_token = m.group("token")

            gitlab_oauth_client = current_app.store.get_oauth_client(
                get_redis_key_from_token(access_token, key_suffix=GL_SUFFIX)
            )

            headers["Authorization"] = f"Bearer {access_token}"
            headers["Gitlab-Access-Token"] = gitlab_oauth_client.access_token

        return headers
