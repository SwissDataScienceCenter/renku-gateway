# -*- coding: utf-8 -*-
#
# Copyright 2018-2019 - Swiss Data Science Center (SDSC)
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

import importlib
import json
import logging
import re
from urllib.parse import urljoin

import jwt
from quart import Blueprint, Response, current_app, request

from app.auth.web import get_valid_token

logger = logging.getLogger(__name__)

blueprint = Blueprint('proxy', __name__)

CHUNK_SIZE = 1024


@blueprint.route('/<path:path>', methods=['GET', 'POST', 'PUT', 'DELETE'])
async def pass_through(path):
    headers = dict(request.headers)

    # Keycloak public key is not defined so error
    if current_app.config['OIDC_PUBLIC_KEY'] is None:
        response = json.dumps("Ooops, something went wrong internally.")
        return Response(response, status=500)

    if 'Host' in headers:
        del headers['Host']

    processor = None
    auth = None

    for key, val in current_app.config['GATEWAY_ENDPOINT_CONFIG'].items():
        p = key.match(path)
        if p:
            try:
                m, _, c = val.get('processor').rpartition('.')
                module = importlib.import_module(m)
                processor = getattr(module, c)(
                    p.group('remaining'), val.get('endpoint')
                )
                if 'auth' in val:
                    m, _, c = val.get('auth').rpartition('.')
                    module = importlib.import_module(m)
                    auth = getattr(module, c)()
                break
            except:
                logger.warning("Error loading processor", exc_info=True)
                return Response(
                    json.dumps({
                        'error': "Error loading processor"
                    }),
                    status=500
                )

    if auth:
        try:
            # validate incomming authentication
            # it can either be in session-cookie or Authorization header
            new_tokens = get_valid_token(headers)
            if new_tokens:
                headers['Authorization'] = "Bearer {}".format(
                    new_tokens.get('access_token')
                )
            if 'Authorization' in headers and 'Referer' in headers:
                allowed = False
                origins = jwt.decode(
                    headers['Authorization'][7:],
                    current_app.config['OIDC_PUBLIC_KEY'],
                    algorithms='RS256',
                    audience=current_app.config['OIDC_CLIENT_ID']
                ).get('allowed-origins')
                for o in origins:
                    if re.match(o.replace("*", ".*"), headers['Referer']):
                        allowed = True
                        break
                if not allowed:
                    return Response(
                        json.dumps({
                            'error':
                                'origin not allowed: {} not matching {}'.
                                format(headers['Referer'], origins)
                        }),
                        status=403
                    )
            if 'Cookie' in headers:
                del headers[
                    'Cookie'
                ]  # don't forward our secret tokens, backend APIs shouldn't expect cookies?

            # auth processors always assume a valid Authorization in header, if any
            headers = auth.process(request, headers)
        except jwt.ExpiredSignatureError:
            return Response(json.dumps({'error': 'token_expired'}), status=401)
        except:
            logger.warning("Error while authenticating request", exc_info=True)
            return Response(
                json.dumps({
                    'error': "Error while authenticating"
                }),
                status=401
            )

    if processor:
        return await processor.process(request, headers)

    else:
        response = json.dumps({'error': "No processor found for this path"})
        return Response(response, status=404)


@blueprint.route('/dummy', methods=['GET'])
async def dummy():
    return 'Dummy works'
