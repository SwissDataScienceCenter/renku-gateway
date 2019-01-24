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
from urllib.parse import urljoin

import quart.flask_patch
import redis
from flask_kvsession import KVSessionExtension
from quart import Quart, Response
from quart_cors import cors
from simplekv.decorator import PrefixDecorator
from simplekv.memory.redisstore import RedisStore

from .auth import gitlab_auth, jupyterhub_auth, web
from .blueprints import graph
from .config import config, load_config
from .gateway import proxy

logging.basicConfig(level=logging.DEBUG)

app = Quart(__name__)
app.config.from_mapping(config)
app = cors(
    app,
    allow_headers=['X-Requested-With'],
    allow_origin=app.config['ALLOW_ORIGIN'],
)

load_config()

if "pytest" in sys.modules:
    from simplekv.memory import DictStore
    store = DictStore()
else:
    store = RedisStore(redis.StrictRedis(host=app.config['REDIS_HOST']))

prefixed_store = PrefixDecorator('sessions_', store)
KVSessionExtension(prefixed_store, app)

url_prefix = app.config['SERVICE_PREFIX']
blueprints = (
    graph.blueprint,
    gitlab_auth.blueprint,
    jupyterhub_auth.blueprint,
    proxy.blueprint,
    web.blueprint,
)


@app.route('/health', methods=['GET'])
async def healthcheck():
    return Response(json.dumps("Up and running"), status=200)


def _join_url_prefix(*parts):
    """Join prefixes."""
    parts = [part.strip('/') for part in parts if part]
    if parts:
        return '/' + '/'.join(parts)


for blueprint in blueprints:
    app.register_blueprint(
        blueprint,
        url_prefix=_join_url_prefix(url_prefix, blueprint.url_prefix),
    )
