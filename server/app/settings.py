# -*- coding: utf-8 -*-
#
# Copyright 2017 - Swiss Data Science Center (SDSC)
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
"""Global settings."""

import os

g = dict()


def settings():
    if not g:
        setup_globals()
    return g


def setup_globals():
    global g
    # g['API_ROOT_URL'] = os.environ.get(
    #     'API_ROOT_URL', 'http://localhost/api/')
    g['KEYCLOAK_URL'] = os.environ.get(
        'KEYCLOAK_URL', 'http://keycloak.renga.local:8080/auth/realms/Renga')
    g['KEYCLOAK_REDIRECT_URL'] = os.environ.get(
        'KEYCLOAK_REDIRECT_URL', 'http://keycloak.renga.local:8080/auth/realms/Renga')
    g['CLIENT_ID'] = os.environ.get(
        'CLIENT_ID', 'demo-client')
    g['CLIENT_SECRET'] = os.environ.get(
        'CLIENT_SECRET', '5294a18e-e784-4e39-a927-ce816c91c83e')
    # g['STORAGE_DEFAULT_BACKEND'] = os.environ.get(
    #     'STORAGE_DEFAULT_BACKEND', 'local')
    # g['DEPLOY_DEFAULT_BACKEND'] = os.environ.get(
    #     'DEPLOY_DEFAULT_BACKEND', 'docker')
    # g['SENTRY_UI_DSN'] = os.environ.get(
    #     'SENTRY_UI_DSN', '')
