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

import json
import re

import jwt
from quart import Blueprint, Response, current_app, redirect, request, session, url_for

from .gitlab_auth import GitlabUserToken
from .web import JWT_ALGORITHM, get_key_for_user

# TODO: We're using a class here only to have a uniform interface
# with GitlabUserToken and JupyterhubUserToken. This should be refactored.
class RenkuCoreAuthHeaders:
    def process(self, request, headers):
        from .. import store

        m = re.search(
            r"bearer (?P<token>.+)", headers.get("Authorization", ""), re.IGNORECASE
        )
        if m:
            access_token = m.group("token")
            decodentoken = jwt.decode(
                access_token,
                current_app.config["OIDC_PUBLIC_KEY"],
                algorithms=JWT_ALGORITHM,
                audience=current_app.config["OIDC_CLIENT_ID"],
            )

            id_token = store.get(get_key_for_user(decodentoken, "kc_id_token")).decode()
            id_dict = json.loads(id_token)

            gl_token = store.get(
                get_key_for_user(decodentoken, "gl_access_token")
            ).decode()

            headers["Renku-user-id"] = id_dict["sub"]
            headers["Renku-user-email"] = id_dict["email"]
            headers["Renku-user-fullname"] = "{} {}".format(
                id_dict["given_name"], id_dict["family_name"]
            )
            headers["Authorization"] = "Bearer {}".format(gl_token)

        else:
            pass

        return headers
