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
"""Gateway logic."""

import logging
import importlib
import json
from flask import request, Response
from urllib.parse import urljoin

from .. import app
from flask_cors import CORS


logger = logging.getLogger(__name__)
CORS(app)

CHUNK_SIZE = 1024


@app.route('/health', methods=['GET'])
def healthcheck():
    return Response(json.dumps("Up and running"), status=200)


@app.route(urljoin(app.config['SERVICE_PREFIX'], '<path:path>'), methods=['GET', 'POST', 'PUT', 'DELETE'])
def pass_through(path):
    headers = dict(request.headers)

    # Keycloak public key is not defined so error
    if app.config['OIDC_PUBLIC_KEY'] is None:
        response = json.dumps("Ooops, something went wrong internally.")
        return Response(response, status=500)

    del headers['Host']

    processor = None
    auth = None

    for key, val in app.config['GATEWAY_ENDPOINT_CONFIG'].items():
        p = key.match(path)
        if p:
            try:
                m, _, c = val.get('processor').rpartition('.')
                module = importlib.import_module(m)
                processor = getattr(module, c)(p.group('remaining'), val.get('endpoint'))
                if 'auth' in val:
                    m, _, c = val.get('auth').rpartition('.')
                    module = importlib.import_module(m)
                    auth = getattr(module, c)()
                break
            except:
                logger.warning("Error loading processor", exc_info=True)

    if auth:
        headers = auth.process(request, headers)

    if processor:
        return processor.process(request, headers)

    else:
        response = json.dumps({'error': "No processor found for this path"})
        return Response(response, status=404)


@app.route(urljoin(app.config['SERVICE_PREFIX'], 'dummy'), methods=['GET'])
def dummy():
    return 'Dummy works'
