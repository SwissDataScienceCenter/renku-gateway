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
import jwt
import logging
import requests
import re

import json
from quart import request, redirect, url_for, session
from urllib.parse import urljoin, urlencode, parse_qs

from oic import rndstr

from .. import app, store
from .web import get_key_for_user, JWT_ALGORITHM

logger = logging.getLogger(__name__)


class JupyterhubUserToken():

    def process(self, request, headers):

        m = re.search(r'bearer (?P<token>.+)', headers.get('Authorization', ''), re.IGNORECASE)
        if m:
            # logger.debug('Authorization header present, token exchange')
            access_token = m.group('token')
            decodentoken = jwt.decode(
                access_token, app.config['OIDC_PUBLIC_KEY'],
                algorithms=JWT_ALGORITHM,
                audience=app.config['OIDC_CLIENT_ID']
            )

            jh_token = store.get(get_key_for_user(decodentoken, 'jh_access_token'))
            headers['Authorization'] = "token {}".format(jh_token.decode())

            # logger.debug('outgoing headers: {}'.format(json.dumps(headers)))
        else:
            # logger.debug("No authorization header, returning empty auth headers")
            headers.pop('Authorization', None)

        return headers

JUPYTERHUB_OAUTH2_PATH = "/hub/api/oauth2"


@app.route(urljoin(app.config['SERVICE_PREFIX'], 'auth/jupyterhub/login'))
def jupyterhub_login():

    state = rndstr()

    session['jupyterhub_state'] = state
    session['jupyterhub_ui_redirect_url'] = request.args.get('redirect_url')

    args = {
        'client_id': app.config['JUPYTERHUB_CLIENT_ID'],
        'response_type': 'code',
        'redirect_uri': app.config['HOST_NAME'] + url_for('jupyterhub_get_tokens'),
        'state': state
    }
    url = app.config['JUPYTERHUB_URL'] + JUPYTERHUB_OAUTH2_PATH + "/authorize"
    login_url = "{}?{}".format(url, urlencode(args))
    response = app.make_response(redirect(login_url))
    return response


@app.route(urljoin(app.config['SERVICE_PREFIX'], 'auth/jupyterhub/token'))
def jupyterhub_get_tokens():

    authorization_parameters = parse_qs(request.query_string.decode())

    if session['jupyterhub_state'] != authorization_parameters['state'][0]:
        return 'Something went wrong while trying to log you in.'

    token_response = requests.post(
        app.config['JUPYTERHUB_URL'] + JUPYTERHUB_OAUTH2_PATH + "/token",
        data={
            'client_id': app.config['JUPYTERHUB_CLIENT_ID'],
            'client_secret': app.config['JUPYTERHUB_CLIENT_SECRET'],
            'state': session['jupyterhub_state'],
            'code': authorization_parameters['code'][0],
            'grant_type': 'authorization_code',
            'redirect_uri': app.config['HOST_NAME'] + url_for('jupyterhub_get_tokens'),
        }
    )

    a = jwt.decode(session['token'], verify=False)
    store.put(get_key_for_user(a, 'jh_access_token'), token_response.json().get('access_token').encode())

    response = app.make_response(redirect(session['jupyterhub_ui_redirect_url']))

    return response


@app.route(urljoin(app.config['SERVICE_PREFIX'], 'auth/jupyterhub/logout'))
def jupyterhub_logout():
    logout_url = app.config['JUPYTERHUB_URL'] + '/hub/logout'
    response = app.make_response(redirect(logout_url))

    return response
