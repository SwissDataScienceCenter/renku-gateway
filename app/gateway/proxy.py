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
import json
import requests
import re
from flask import request, Response
from urllib.parse import urljoin, quote_plus

from app.helpers.gitlab_parsers import parse_project
from app.helpers.gitlab_user_utils import get_or_create_gitlab_user
from .. import app
from ..auth.web import swapped_token
from flask_cors import CORS
import jwt
from jwt.exceptions import ExpiredSignatureError
from functools import wraps


logger = logging.getLogger(__name__)
CORS(app)

CHUNK_SIZE = 1024

# TODO use token
# def with_tokens(f):
#      """Function decorator to ensure we have OIDC tokens"""
#      return 0
def authorize():
    def decorator(f):
        @wraps(f)
        def decorated_function(path):
            headers = dict(request.headers)
            if 'Authorization' in headers:
                # logger.debug('Authorization header present, sudo token exchange')
                # logger.debug('outgoing headers: {}'.format(json.dumps(headers))

                # TODO: Use regular expressions to extract the token from the header
                access_token = headers.get('Authorization')[7:]
                del headers['Authorization']
                headers['Private-Token'] = app.config['GITLAB_PASS']

                # Decode token to get user id
                # TODO: What happens if the validation of the token fails for other reasons?
                try:
                    decodentoken = jwt.decode(access_token, app.config['OIDC_PUBLIC_KEY'], algorithms='RS256',
                                                audience=app.config['OIDC_CLIENT_ID'])
                except ExpiredSignatureError:
                    return Response('Access token expired', 401)

                headers['Sudo'] = get_or_create_gitlab_user(decodentoken)

            else:
                # logger.debug("No authorization header, returning empty auth headers")
                headers.pop('Sudo', None)

            return f(path, headers=headers)
        return decorated_function
    return decorator

@app.route('/health', methods=['GET'])
def healthcheck():
    return Response(json.dumps("Up and running"), status=200)


# @app.route(urljoin(app.config['SERVICE_PREFIX'], 'projects'), methods=['GET'])
# def map_project():
#     logger.debug('projects controller')
#
#     headers = dict(request.headers)
#
#     del headers['Host']
#
#     auth_headers = authorize(headers)
#     if auth_headers!=[] :
#
#         project_url = app.config['GITLAB_URL'] + "/api/v4/projects"
#         project_response = requests.request(request.method, project_url, headers=headers, data=request.data, stream=True, timeout=300)
#         projects_list = project_response.json()
#         return_project = json.dumps([parse_project(headers, x) for x in projects_list])
#
#         return Response(return_project, project_response.status_code)
#
#     else:
#         response = json.dumps("No authorization header found")
#         return Response(response, status=401)


@app.route(urljoin(app.config['SERVICE_PREFIX'], '<path:path>'), methods=['GET', 'POST', 'PUT', 'DELETE'])
@swapped_token()
@authorize()
def pass_through(path, headers=None):

    # Gitlab has routes where the resource identifier can include slashes
    # which must be url-encoded. We list these routes individually and re-encode
    # slashes which have been unencoded by uWSGI.
    path = urlencode_paths(path)

    # Keycloak public key is not defined so error
    if app.config['OIDC_PUBLIC_KEY'] is None:
        response = json.dumps("Ooops, something went wrong internally.")
        return Response(response, status=500)

    del headers['Host']

    # TODO: The actual backend service responsible for a given request should not be specified as part of the URL,
    # TODO: i.e. the client should not care if it is storage, gitlab, etc which is going to serve its request.
    # TODO: This switch should be resource dependent.

    # TODO: This needs to be fixed as the gateway is now occupying all /api/... routes.
    if path.startswith('storage'):
        url = app.config['RENKU_ENDPOINT'] + "/api/" + path
    else:
        url = app.config['GITLAB_URL'] + "/api/" + path

    # Forward request to backend service
    response = requests.request(
        request.method,
        url,
        headers=headers,
        params=request.args,
        data=request.data,
        stream=True,
        timeout=300
    )
    # logger.debug('Response: {}'.format(response.status_code))
    return Response(generate(response), response.status_code)


SPECIAL_ROUTE_RULES = [
    {
        'before': 'repository/files/',
        'after': '/raw'
    }
]

SPECIAL_ROUTE_REGEXES = [
    '(.*)({before})(.*)({after})(.*)'.format(before=rule['before'], after=rule['after']) for rule in SPECIAL_ROUTE_RULES
]


def urlencode_paths(path):
    """Urlencode some paths before forwarding requests."""

    for rule_regex in SPECIAL_ROUTE_REGEXES:
        m = re.search(rule_regex, path)
        if m:
            return '{leading}{before}{match}{after}{trailing}'.format(
                leading=m.group(1),
                before=m.group(2),
                match=quote_plus(m.group(3)),
                after=m.group(4),
                trailing=m.group(5)
            )
    return path


@app.route(urljoin(app.config['SERVICE_PREFIX'], 'dummy'), methods=['GET'])
def dummy():
    return 'Dummy works'


def generate(response):
    for c in response.iter_lines():
        yield c + "\r".encode()
    return(response)
