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
"""Proxy controller."""

import logging
import json
import requests
from flask import request, Response

from app.helpers.gitlab_parsers import parse_project
from .. import app
from ..auth.web import swapped_token
from flask_cors import CORS
import jwt


logger = logging.getLogger(__name__)
CORS(app)

CHUNK_SIZE = 1024

# TODO use token
# def with_tokens(f):
#      """Function decorator to ensure we have OIDC tokens"""
#      return 0


@app.route('/api/projects', methods=['GET'])
def map_project():
    logger.debug('projects controller')

    headers = dict(request.headers)

    del headers['Host']

    auth_headers = authorize(headers)
    if auth_headers!=[] :

        project_url = app.config['GITLAB_URL'] + "/api/v4/projects"
        project_response = requests.request(request.method, project_url, headers=headers, data=request.data, stream=True, timeout=300)
        projects_list = project_response.json()
        return_project = json.dumps([parse_project(headers, x) for x in projects_list])

        return Response(return_project, project_response.status_code)

    else:
        response = json.dumps("No authorization header found")
        return Response(response, status=401)


@app.route('/api/<path:path>', methods=['GET', 'POST', 'PUT', 'DELETE'])
@swapped_token()
def pass_through(path):
    # Keycloak public key is not defined so error

    if app.config['OIDC_PUBLIC_KEY'] is None:
        response = json.dumps("Ooops, something went wrong internally.")

        return Response(response, status=500)

    logger.debug(path)

    headers = dict(request.headers)

    del headers['Host']

    if path.startswith('gitlab'):
        # do the gitlab token swapping
        auth_headers = authorize(headers, g)
        url = g['GITLAB_URL'] + "/api/" + path

        if not auth_headers:
            response = json.dumps("No authorization header found")
            return Response(response, status=401)

    elif path.startswith('storage'):
        url = g['RENKU_ENDPOINT'] + "/api/" + path

    auth_headers = authorize(headers)

    if auth_headers!=[] :
         # Respond to requester
       #  url = app.config['GITLAB_URL'] + "/api/" + path
         response = requests.request(request.method, url, headers=headers, data=request.data, stream=True, timeout=300)
         logger.debug('Response: {}'.format(response.status_code))
         return Response(generate(response), response.status_code)

    else:
        return Response("Not Found", status=404)






@app.route('/api/dummy', methods=['GET'])
def dummy():
    return 'Dummy works'


def authorize(headers):
    if 'Authorization' in headers:

        access_token = headers.get('Authorization')[7:]
        del headers['Authorization']
        headers['Private-Token'] = app.config['GITLAB_PASS']

        # Decode token to get user id
        decodentoken = jwt.decode(access_token, app.config['OIDC_PUBLIC_KEY'], algorithms='RS256', audience=app.config['OIDC_CLIENT_ID'])
        id = (decodentoken['preferred_username'])
        headers['Sudo'] = id
        headers['Private-Token'] = app.config['GITLAB_PASS']
        logger.debug(headers)

        return headers

    else:
        return []


def generate(response):
    for c in response.iter_lines():
        logger.debug(c)
        yield c + "\r".encode()
    return(response)
