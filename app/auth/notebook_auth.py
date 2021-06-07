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
"""Add the headers for the Renku notebooks service."""
import json
import re
from base64 import b64encode

from flask import current_app

from .gitlab_auth import GL_SUFFIX
from .utils import get_redis_key_from_token
from .web import KC_SUFFIX

# TODO: This is a temporary implementation of the header interface defined in #404
# TODO: that allowes first clients to transition.


def get_git_credentials_header(git_oauth_clients):
    """
    Create the git authentication header as defined in #406
    (https://github.com/SwissDataScienceCenter/renku-gateway/issues/406)
    given a list of GitLab/GitHub oauth client objects.
    """

    git_credentials = {
        client.provider_app.base_url: {
            # TODO: add this information to the provider_app and read it from there.
            "provider": "GitLab",
            "AuthorizationHeader": f"bearer {client.access_token}",
        }
        for client in git_oauth_clients
    }
    git_credentials_json = json.dumps(git_credentials)
    return b64encode(git_credentials_json.encode()).decode("ascii")


class NotebookAuthHeaders:
    def process(self, request, headers):

        m = re.search(
            r"bearer (?P<token>.+)", headers.get("Authorization", ""), re.IGNORECASE
        )
        if m:
            access_token = m.group("token")

            keycloak_oidc_client = current_app.store.get_oauth_client(
                get_redis_key_from_token(access_token, key_suffix=KC_SUFFIX)
            )
            gitlab_oauth_client = current_app.store.get_oauth_client(
                get_redis_key_from_token(access_token, key_suffix=GL_SUFFIX)
            )

            headers["Renku-Auth-Access-Token"] = access_token
            headers["Renku-Auth-Id-Token"] = keycloak_oidc_client.token["id_token"]
            headers["Renku-Auth-Git-Credentials"] = get_git_credentials_header(
                [gitlab_oauth_client]
            )

        return headers
