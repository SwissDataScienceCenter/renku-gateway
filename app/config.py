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
"""Global settings."""

import os
import sys
import warnings


ANONYMOUS_SESSIONS_ENABLED = (
    os.environ.get("ANONYMOUS_SESSIONS_ENABLED", "false") == "true"
)

HOST_NAME = os.environ.get("HOST_NAME", "http://gateway.renku.build")

if "GATEWAY_SECRET_KEY" not in os.environ and "pytest" not in sys.modules:
    warnings.warn(
        "The environment variable GATEWAY_SECRET_KEY is not set. "
        "It is mandatory for securely signing session cookie."
    )
    sys.exit(2)
SECRET_KEY = os.environ.get("GATEWAY_SECRET_KEY")

SESSION_COOKIE_HTTPONLY = True
SESSION_COOKIE_SECURE = HOST_NAME.startswith("https")

ALLOW_ORIGIN = os.environ.get("GATEWAY_ALLOW_ORIGIN", "").split(",")

REDIS_HOST = os.environ.get("GATEWAY_REDIS_HOST", "renku-gw-redis")

GITLAB_URL = os.environ.get("GITLAB_URL", "http://gitlab.renku.build")

GITLAB_CLIENT_ID = os.environ.get("GITLAB_CLIENT_ID", "renku-ui")
GITLAB_CLIENT_SECRET = os.environ.get("GITLAB_CLIENT_SECRET")
if not GITLAB_CLIENT_SECRET:
    warnings.warn(
        "The environment variable GITLAB_CLIENT_SECRET is not set."
        "It is mandatory for Gitlab login."
    )

JUPYTERHUB_URL = os.environ.get("JUPYTERHUB_URL", "{}/jupyterhub".format(HOST_NAME))
if ANONYMOUS_SESSIONS_ENABLED:
    JUPYTERHUB_TMP_URL = "{}-tmp".format(JUPYTERHUB_URL)

JUPYTERHUB_CLIENT_ID = os.environ.get("JUPYTERHUB_CLIENT_ID", "gateway")
JUPYTERHUB_CLIENT_SECRET = os.environ.get("JUPYTERHUB_CLIENT_SECRET")
if not JUPYTERHUB_CLIENT_SECRET:
    warnings.warn(
        "The environment variable JUPYTERHUB_CLIENT_SECRET is not set. "
        "It is mandatory for JupyterHub login."
    )

WEBHOOK_SERVICE_HOSTNAME = os.environ.get(
    "WEBHOOK_SERVICE_HOSTNAME", "http://renku-graph-webhooks-service"
)

SPARQL_ENDPOINT = os.environ.get(
    "SPARQL_ENDPOINT", "http://localhost:3030/renku/sparql"
)
SPARQL_USERNAME = os.environ.get("SPARQL_USERNAME", "admin")
SPARQL_PASSWORD = os.environ.get("SPARQL_PASSWORD", "admin")

KEYCLOAK_URL = os.environ.get("KEYCLOAK_URL")
if not KEYCLOAK_URL:
    warnings.warn(
        "The environment variable KEYCLOAK_URL is not set. "
        "It is necessary because Keycloak acts as identity provider for Renku."
    )
KEYCLOAK_REALM = os.environ.get("KEYCLOAK_REALM", "Renku")
OIDC_ISSUER = "{}/auth/realms/{}".format(KEYCLOAK_URL, KEYCLOAK_REALM)
OIDC_CLIENT_ID = os.environ.get("OIDC_CLIENT_ID", "renku")
OIDC_CLIENT_SECRET = os.environ.get("OIDC_CLIENT_SECRET")
if not OIDC_CLIENT_SECRET:
    warnings.warn(
        "The environment variable OIDC_CLIENT_SECRET is not set. "
        "It is mandatory for OpenId-Connect login."
    )

SERVICE_PREFIX = os.environ.get("GATEWAY_SERVICE_PREFIX", "/")
