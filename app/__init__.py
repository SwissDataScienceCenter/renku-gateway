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
import sys
import os

import quart.flask_patch
import redis
from flask_kvsession import KVSessionExtension
from quart import Quart, Response
from quart_cors import cors
from simplekv.decorator import PrefixDecorator
from simplekv.memory.redisstore import RedisStore

from .auth import gitlab_auth, jupyterhub_auth, web
from .blueprints import gitlab, graph, jupyterhub, notebooks, webhooks
from .config import config

# Wait for the VS Code debugger to attach if requested
VSCODE_DEBUG = os.environ.get('VSCODE_DEBUG') == "1"
print("{}:{}".format(os.environ.get('VSCODE_DEBUG'), VSCODE_DEBUG))
if VSCODE_DEBUG:
    import ptvsd

    # 5678 is the default attach port in the VS Code debug configurations
    print("Waiting for debugger attach")
    ptvsd.enable_attach(address=('localhost', 5678), redirect_output=True)
    ptvsd.wait_for_attach()
    breakpoint()

logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger(__name__)


app = Quart(__name__)
app.config.from_mapping(config)
app = cors(
    app,
    allow_headers=['X-Requested-With'],
    allow_origin=app.config['ALLOW_ORIGIN'],
)

if "pytest" in sys.modules:
    from simplekv.memory import DictStore
    store = DictStore()
else:
    store = RedisStore(redis.StrictRedis(host=app.config['REDIS_HOST']))

prefixed_store = PrefixDecorator('sessions_', store)
KVSessionExtension(prefixed_store, app)

url_prefix = app.config['SERVICE_PREFIX']
blueprints = (
    gitlab_auth.blueprint,
    jupyterhub_auth.blueprint,
    web.blueprint,
    graph.blueprint,
    notebooks.blueprint,
    jupyterhub.blueprint,
    webhooks.blueprint,
    gitlab.blueprint,
)


@app.route('/health', methods=['GET'])
async def healthcheck():
    return Response(json.dumps("Up and running"), status=200)


def _join_url_prefix(*parts):
    """Join prefixes."""
    parts = [part.strip('/') for part in parts if part and part.strip('/')]
    if parts:
        return '/' + '/'.join(parts)


for blueprint in blueprints:
    app.register_blueprint(
        blueprint,
        url_prefix=_join_url_prefix(url_prefix, blueprint.url_prefix),
    )
