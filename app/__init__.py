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
"""Quart initialization."""

import json
import logging
import os
import re
import sys

import jwt
import requests
import sentry_sdk
from flask import Flask, Response, current_app, request
from flask_cors import CORS
from flask_kvsession import KVSessionExtension
from redis.sentinel import Sentinel
from sentry_sdk.integrations.flask import FlaskIntegration
from simplekv.decorator import PrefixDecorator
from simplekv.memory.redisstore import RedisStore

from . import config
from .auth import cli_auth, gitlab_auth, renku_auth, web, notebook_auth
from .auth.oauth_redis import OAuthRedis
from .auth.utils import decode_keycloak_jwt

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
app.logger.setLevel(logging.INFO)
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

CORS(
    app,
    allow_headers=["X-Requested-With"],
    allow_origin=app.config["ALLOW_ORIGIN"],
)

url_prefix = app.config["SERVICE_PREFIX"]
blueprints = (
    cli_auth.blueprint,
    gitlab_auth.blueprint,
    web.blueprint,
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
        # We use the same store for sessions.
        session_store = PrefixDecorator("sessions_", RedisStore(current_app.store))
        KVSessionExtension(session_store, app)


@app.route("/", methods=["GET"])
def auth():
    if "auth" not in request.args:
        return Response("", status=200)

    auths = {
        "gitlab": gitlab_auth.GitlabUserToken,
        "renku": renku_auth.RenkuCoreAuthHeaders,
        "notebook": notebook_auth.NotebookAuthHeaders,
        "cli-gitlab": cli_auth.RenkuCLIGitlabAuthHeaders,
    }

    # Keycloak public key is not defined so error
    if current_app.config["OIDC_PUBLIC_KEY"] is None:
        response = json.dumps("Ooops, something went wrong internally.")
        return Response(response, status=500)

    auth_arg = request.args.get("auth")
    headers = dict(request.headers)

    try:
        auth = auths[auth_arg]()

        # validate incoming authentication
        # it can either be in session-cookie or Authorization header
        new_token = web.get_valid_token(headers)
        if new_token:
            headers["Authorization"] = f"Bearer {new_token}"

        if "Authorization" in headers and "Referer" in headers:
            allowed = False
            origins = decode_keycloak_jwt(token=headers["Authorization"][7:]).get(
                "allowed-origins"
            )
            for o in origins:
                if re.match(o.replace("*", ".*"), headers["Referer"]):
                    allowed = True
                    break
            if not allowed:
                return Response(
                    json.dumps(
                        {
                            "error": "origin not allowed: {} not matching {}".format(
                                headers["Referer"], origins
                            )
                        }
                    ),
                    status=403,
                )

        # auth processors always assume a valid Authorization in header, if any
        headers = auth.process(request, headers)
    except jwt.ExpiredSignatureError:
        current_app.logger.warning(
            f"Error while authenticating request, token expired. Target: {auth_arg}",
            exc_info=True,
        )
        message = {
            "error": "authentication",
            "message": "token expired",
            "target": auth_arg,
        }
        return Response(json.dumps(message), status=401)
    except AttributeError as error:
        if "access_token" in str(error):
            current_app.logger.warning(
                (
                    "Error while authenticating request, can't "
                    f"refresh access token. Target: {auth_arg}"
                ),
                exc_info=True,
            )
            message = {
                "error": "authentication",
                "message": "can't refresh access token",
                "target": auth_arg,
                "exception": str(error),
            }
            return Response(json.dumps(message), status=401)
        raise
    except Exception as error:
        current_app.logger.warning(
            f"Error while authenticating request, unknown. Target: {auth_arg}",
            exc_info=True,
        )
        message = {
            "error": "authentication",
            "message": "unknown",
            "target": auth_arg,
            "exception": str(error),
        }
        return Response(json.dumps(message), status=401)

    if (
        "anon-id" not in request.cookies
        and request.headers.get("X-Requested-With", "") == "XMLHttpRequest"
        and request.headers["X-Forwarded-Uri"] == "/api/user"
        and "Authorization" not in headers
    ):
        resp = Response(
            json.dumps({"message": "401 Unauthorized"}),
            content_type="application/json",
            status=401,
        )
        return resp

    return Response(
        json.dumps("Up and running"),
        headers=headers,
        status=200,
    )


@app.route("/health", methods=["GET"])
def healthcheck():
    return Response(json.dumps("Up and running"), status=200)


def _join_url_prefix(*parts):
    """Join prefixes."""
    parts = [part.strip("/") for part in parts if part and part.strip("/")]
    if parts:
        return "/" + "/".join(parts)


for blueprint in blueprints:
    app.register_blueprint(
        blueprint,
        url_prefix=_join_url_prefix(url_prefix, blueprint.url_prefix),
    )


@app.before_request
def load_public_key():
    if current_app.config.get("OIDC_PUBLIC_KEY"):
        return

    current_app.logger.info(
        "Obtaining public key from {}".format(current_app.config["OIDC_ISSUER"])
    )

    raw_key = requests.get(current_app.config["OIDC_ISSUER"]).json()["public_key"]
    current_app.config[
        "OIDC_PUBLIC_KEY"
    ] = "-----BEGIN PUBLIC KEY-----\n{}\n-----END PUBLIC KEY-----".format(raw_key)

    current_app.logger.info(current_app.config["OIDC_PUBLIC_KEY"])
