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

# This setting can force tokens to be refreshed in case
# they are issued with a too long or unlimited lifetime.
# This is currently the case for BOTH JupyterHub and GitLab!

# Note that for a clean "client side token expiration" we will
# need https://gitlab.com/gitlab-org/gitlab/-/issues/17259 to be
# fixed and the implementation of JupyterHub as an OAuth2 provider
# completed.
MAX_ACCESS_TOKEN_LIFETIME = os.environ.get("MAX_ACCESS_TOKEN_LIFETIME", 3600 * 24)

ANONYMOUS_SESSIONS_ENABLED = (
    os.environ.get("ANONYMOUS_SESSIONS_ENABLED", "false") == "true"
)

HOST_NAME = os.environ.get("HOST_NAME", "http://gateway.renku.build")

if "GATEWAY_SECRET_KEY" not in os.environ and "pytest" not in sys.modules:
    warnings.warn(
        "The environment variable GATEWAY_SECRET_KEY is not set. "
        "It is mandatory for securely signing session cookies and "
        "encrypting REDIS content."
    )
    sys.exit(2)
SECRET_KEY = os.environ.get("GATEWAY_SECRET_KEY")

SESSION_COOKIE_HTTPONLY = True
SESSION_COOKIE_SECURE = HOST_NAME.startswith("https")

ALLOW_ORIGIN = os.environ.get("GATEWAY_ALLOW_ORIGIN", "").split(",")

REDIS_HOST = os.environ.get("REDIS_HOST", "renku-redis")
REDIS_IS_SENTINEL = os.environ.get("REDIS_IS_SENTINEL", "") == "true"
try:
    REDIS_PASSWORD = os.environ["REDIS_PASSWORD"]
except KeyError:
    warnings.warn(
        "No redis password found. Are you sure you don't need one to access redis?"
    )
REDIS_DB = os.environ.get("REDIS_DB", "0")

CLI_CLIENT_ID = os.environ.get("CLI_CLIENT_ID", "renku-cli")
CLI_CLIENT_SECRET = os.environ.get("CLI_CLIENT_SECRET", "")
if not CLI_CLIENT_SECRET:
    warnings.warn(
        "The environment variable CLI_CLIENT_SECRET is not set."
        "It is mandatory for CLI login."
    )

CLI_LOGIN_TIMEOUT = int(os.environ.get("CLI_LOGIN_TIMEOUT", 300))

GITLAB_URL = os.environ.get("GITLAB_URL", "http://gitlab.renku.build")
GITLAB_CLIENT_ID = os.environ.get("GITLAB_CLIENT_ID", "renku-ui")
GITLAB_CLIENT_SECRET = os.environ.get("GITLAB_CLIENT_SECRET")
if not GITLAB_CLIENT_SECRET:
    warnings.warn(
        "The environment variable GITLAB_CLIENT_SECRET is not set."
        "It is mandatory for Gitlab login."
    )

WEBHOOK_SERVICE_HOSTNAME = os.environ.get(
    "WEBHOOK_SERVICE_HOSTNAME", "http://renku-graph-webhooks-service"
)

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

OLD_GITLAB_LOGOUT = os.environ.get("OLD_GITLAB_LOGOUT", "") == "true"

LOGOUT_GITLAB_UPON_RENKU_LOGOUT = (
    os.environ.get("LOGOUT_GITLAB_UPON_RENKU_LOGOUT", "") == "true"
)
