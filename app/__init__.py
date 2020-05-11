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
import re
import sys
import os

import jwt
import quart.flask_patch
import redis
import requests
from flask_kvsession import KVSessionExtension
from quart import Quart, Response, current_app, request
from quart_cors import cors
from simplekv.decorator import PrefixDecorator
from simplekv.memory.redisstore import RedisStore

from . import config
from .auth import gitlab_auth, jupyterhub_auth, web

# Wait for the VS Code debugger to attach if requested
VSCODE_DEBUG = os.environ.get("VSCODE_DEBUG") == "1"
if VSCODE_DEBUG:
    import ptvsd

    # 5678 is the default attach port in the VS Code debug configurations
    print("Waiting for debugger attach")
    ptvsd.enable_attach(address=("localhost", 5678), redirect_output=True)
    ptvsd.wait_for_attach()
    breakpoint()


app = Quart(__name__)

# We activate all log levels and prevent logs from showing twice.
app.logger.setLevel(logging.DEBUG)
app.logger.propagate = False

app.config.from_object(config)

app = cors(
    app, allow_headers=["X-Requested-With"], allow_origin=app.config["ALLOW_ORIGIN"],
)

if "pytest" in sys.modules:
    from simplekv.memory import DictStore

    store = DictStore()
else:
    store = RedisStore(redis.StrictRedis(host=app.config["REDIS_HOST"]))

prefixed_store = PrefixDecorator("sessions_", store)
KVSessionExtension(prefixed_store, app)

url_prefix = app.config["SERVICE_PREFIX"]
blueprints = (
    gitlab_auth.blueprint,
    jupyterhub_auth.blueprint,
    web.blueprint,
)


@app.route("/", methods=["GET"])
async def auth():
    if "auth" not in request.args:
        return Response("", status=200)

    from .auth.gitlab_auth import GitlabUserToken
    from .auth.jupyterhub_auth import JupyterhubUserToken
    from .auth.renku_auth import RenkuCoreAuthHeaders

    auths = {
        "gitlab": GitlabUserToken,
        "jupyterhub": JupyterhubUserToken,
        "renku": RenkuCoreAuthHeaders,
    }

    auth = auths[request.args.get("auth")]()
    headers = dict(request.headers)

    # Keycloak public key is not defined so error
    if current_app.config["OIDC_PUBLIC_KEY"] is None:
        response = json.dumps("Ooops, something went wrong internally.")
        return Response(response, status=500)

    try:
        # validate incomming authentication
        # it can either be in session-cookie or Authorization header
        new_tokens = web.get_valid_token(headers)
        if new_tokens:
            headers["Authorization"] = "Bearer {}".format(
                new_tokens.get("access_token")
            )
        if "Authorization" in headers and "Referer" in headers:
            allowed = False
            origins = jwt.decode(
                headers["Authorization"][7:],
                current_app.config["OIDC_PUBLIC_KEY"],
                algorithms="RS256",
                audience=current_app.config["OIDC_CLIENT_ID"],
            ).get("allowed-origins")
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
        return Response(json.dumps({"error": "token_expired"}), status=401)
    except:
        current_app.logger.warning("Error while authenticating request", exc_info=True)
        return Response(json.dumps({"error": "Error while authenticating"}), status=401)

    return Response(json.dumps("Up and running"), headers=headers, status=200,)


@app.route("/health", methods=["GET"])
async def healthcheck():
    return Response(json.dumps("Up and running"), status=200)


def _join_url_prefix(*parts):
    """Join prefixes."""
    parts = [part.strip("/") for part in parts if part and part.strip("/")]
    if parts:
        return "/" + "/".join(parts)


for blueprint in blueprints:
    app.register_blueprint(
        blueprint, url_prefix=_join_url_prefix(url_prefix, blueprint.url_prefix),
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
