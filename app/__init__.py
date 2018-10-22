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
"""Quart initialization."""

import logging
import redis
import sys
import quart.flask_patch
from flask_kvsession import KVSessionExtension
from simplekv.decorator import PrefixDecorator
from simplekv.memory.redisstore import RedisStore
from quart import Quart
from quart_cors import cors

from .config import config, load_config

logging.basicConfig(level=logging.DEBUG)

app = Quart(__name__)

for key in config.keys():
    app.config[key] = config[key]

app = cors(app, allow_headers=['X-Requested-With'], allow_origin=app.config['ALLOW_ORIGIN'])

load_config()

if "pytest" in sys.modules:
    from mockredis import mock_strict_redis_client
    store = RedisStore(mock_strict_redis_client())
else:
    store = RedisStore(redis.StrictRedis(host=app.config['REDIS_HOST']))

prefixed_store = PrefixDecorator('sessions_', store)
KVSessionExtension(prefixed_store, app)

from .gateway import proxy
from .auth import web
from .auth.gitlab_auth import gitlab_login, gitlab_get_tokens, gitlab_logout
from .auth.jupyterhub_auth import jupyterhub_login, jupyterhub_get_tokens, jupyterhub_logout
