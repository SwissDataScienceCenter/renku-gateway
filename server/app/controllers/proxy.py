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
"""Proxy controller."""

import logging
import json
import requests
from flask import request, session, Response

from .. import app
#from ..keycloak import require_tokens
from ..settings import settings

logger = logging.getLogger(__name__)

CHUNK_SIZE = 1024


# TODO use token
def with_tokens(f):
     """Function decorator to ensure we have OIDC tokens"""
     return 0

@app.route('/api/<path:path>', methods=['GET', 'POST', 'PUT', 'DELETE','OPTIONS'])
#@require_tokens
def pass_through(path):
    logger.debug(path)
    logger.debug(request)
    print("********************************")
    g = settings()

 #   access_token = 'Bearer {}'.format(session['tokens']['access_token'])

 #   url = '{base}{path}'.format(base=g['API_ROOT_URL'], path=path)
  #  if request.query_string:
       # url += '?{}'.format(request.query_string.decode())
    url = "http://gitlab.renga.local" + "/api/v4/"+ path

    headers = dict(request.headers)
    print(headers)
    del headers['Host']
   # del headers['Cookie']


    sudotoken = "yxsp5Ut32KR66E-HY7oc"
    #   if 'Authorization' not in headers:
 #       headers['Authorization'] = access_token
    headers['Private-Token'] =  sudotoken
    headers['Sudo'] = "demo"



    logger.debug('Method: {}'.format(request.method))
    logger.debug('URL: {}'.format(url))
    logger.debug('Headers: {}'.format(headers))
    logger.debug('Data (maybe truncated): {}'.format(request.data[0:1000]))

    response = requests.request(request.method, url, headers=headers, data=request.data, stream=True, timeout=300)

    logger.debug('Response: {}'.format(response.status_code))

    def generate():
        for c in response.iter_lines():
            logger.debug(c)
            yield c + "\r".encode()

    return Response(generate(), response.status_code)


# @app.route('/webproxy', methods=['GET'])
# #@require_tokens
# def webproxy():
#     """Simplest possible webproxy to avoid CORS problems when loading external datasets in the UI."""
#     logger.debug(request)
#
#     url = request.headers.get('fileUrl')
#     logger.debug('resolving URL: {}'.format(url))
#     response = requests.request('GET', url, stream=True, timeout=3000)
#
#     def generate():
#         for c in response.iter_content(1024):
#             yield c
#
#     headers = dict(response.headers)
#
#     # TODO: We go unencoded for the moment (otherwise zip-files etc. are broken)
#     try:
#         del headers["Transfer-Encoding"]
#     except KeyError:
#         pass
#     try:
#         del headers["Content-Encoding"]
#     except KeyError:
#         pass
#
#     return Response(generate(), response.status_code,  headers=headers)

# @app.route('/download', methods=['GET'])
# #@require_tokens
# def download():
#     logger.debug(request)
#
#     g = settings()
#
#     access_token = 'Bearer {}'.format(session['tokens']['access_token'])
#
#     url = '{base}{path}'.format(base=g['API_ROOT_URL'], path='storage/authorize/read')
#     headers = dict(request.headers)
#     del headers['Host']
#     del headers['Cookie']
#     headers['Authorization'] = access_token
#     headers['Content-Type'] = 'text/json'
#
#     logger.debug('URL: {}'.format(url))
#     logger.debug('Headers: {}'.format(headers))
#     logger.debug('Data: {}'.format(json.dumps({"resource_id": int(request.args.get('id')), "request_type": "read_file"})))
#
#     response = requests.request('POST', url, headers=headers, data=json.dumps({"resource_id": int(request.args.get('id')), "request_type": "read_file"}), timeout=300)
#
#     if response.status_code == 200:
#         logger.debug('Response: {}'.format(response.status_code))
#         logger.debug('Response: {}'.format(response.json()))
#
#         url = '{base}{path}'.format(base=g['API_ROOT_URL'], path='storage/io/read')
#         token = "Bearer {}".format(response.json().get('access_token'))
#         headers['Authorization'] = token
#
#         response = requests.request('GET', url, headers=headers, stream=True, timeout=300)
#
#     def generate():
#         for c in response.iter_content(CHUNK_SIZE):
#             yield c
#
#     resp = Response(generate(), response.status_code)
#     resp.headers['Content-Disposition'] = 'attachment; filename={}'.format(request.args.get('name', 'untitled'))
#
#     return resp
