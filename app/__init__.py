# -*- coding: utf-8 -*-
#
# Copyright 2017-2019 - Swiss Data Science Center (SDSC)
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
from datetime import timedelta
import json
import logging
import os
import sys

import sentry_sdk
from flask import Flask, Response, current_app, session
from redis.sentinel import Sentinel
from sentry_sdk.integrations.flask import FlaskIntegration

from . import config
from .forward_auth import forward_auth
from .web import gitlab, keycloak
from .common.oauth_redis import OAuthRedis
from .common.utils import load_public_key

# Wait for the VS Code debugger to attach if requested
# TODO: try using debugpy instead of ptvsd to avoid noreload limitations

VSCODE_DEBUG = os.environ.get("VSCODE_DEBUG") == "1"
if VSCODE_DEBUG:
    import debugpy

    print("Waiting for a debugger to attach")
    # 5678 is the default attach port in the VS Code debug configurations
    debugpy.listen(("localhost", 5678))
    debugpy.wait_for_client()
    breakpoint()

app = Flask(__name__)

# We activate all log levels and prevent logs from showing twice.
app.logger.setLevel(logging.DEBUG)
app.logger.propagate = False

# Initialize Sentry when required
if os.environ.get("SENTRY_ENABLED", "").lower() == "true":
    try:
        sentry_sdk.init(
            dsn=os.environ.get("SENTRY_DSN"),
            integrations=[FlaskIntegration()],
            environment=os.environ.get("SENTRY_ENVIRONMENT"),
        )
    except Exception:
        app.logger.warning("Error while trying to initialize Sentry", exc_info=True)

app.config.from_object(config)
url_prefix = app.config["SERVICE_PREFIX"]

app.register_blueprint(
    keycloak.blueprint,
    url_prefix=os.path.join(url_prefix, keycloak.blueprint.url_prefix),
)
app.register_blueprint(
    gitlab.blueprint,
    url_prefix=os.path.join(url_prefix, gitlab.blueprint.url_prefix),
)
app.register_blueprint(
    forward_auth.blueprint,
    url_prefix=forward_auth.blueprint.url_prefix,
)


@app.before_request
def setup_redis_client():
    """Set up a redis connection to the master by querying sentinel."""

    if "pytest" not in sys.modules:

        if current_app.config["REDIS_IS_SENTINEL"]:
            sentinel = Sentinel(
                [(current_app.config["REDIS_HOST"], current_app.config["REDIS_PORT"])],
                sentinel_kwargs={"password": current_app.config["REDIS_PASSWORD"]},
            )
            host, port = sentinel.discover_master(
                current_app.config["REDIS_MASTER_SET"]
            )
            current_app.logger.debug(f"Discovered redis master at {host}:{port}")
        else:
            (host, port) = (
                current_app.config["REDIS_HOST"],
                current_app.config["REDIS_PORT"],
            )

        # Set up the redis store for tokens
        current_app.store = OAuthRedis(
            hex_key=current_app.config["SECRET_KEY"],
            host=host,
            password=current_app.config["REDIS_PASSWORD"],
            db=current_app.config["REDIS_DB"],
        )


@app.before_request
def get_public_key():
    load_public_key(timedelta(hours=1))


@app.route("/health", methods=["GET"])
def healthcheck():
    return Response(json.dumps("Up and running"), status=200)


@app.before_request
def make_session_permanent():
    session.permanent = True
