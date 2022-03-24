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
"""Implement GitLab authentication workflow."""

from flask import (
    current_app,
)

from ..common.utils import build_redis_key, GL_SUFFIX


# Note: GitLab oauth tokens do NOT expire per default
# See https://gitlab.com/gitlab-org/gitlab/-/issues/21745
# The documentation about this is wrong
# (https://docs.gitlab.com/ce/api/oauth2.html#web-application-flow)


def add_auth_headers(sub, headers):
    redis_key = build_redis_key(sub, key_suffix=GL_SUFFIX)
    gitlab_oauth_client = current_app.store.get_oauth_client(redis_key)

    if gitlab_oauth_client:
        gitlab_access_token = gitlab_oauth_client.access_token
        headers["Authorization"] = f"Bearer {gitlab_access_token}"
    return headers
