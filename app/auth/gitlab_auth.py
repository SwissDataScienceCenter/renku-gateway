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
import urllib

import json
import re
from oic.oauth2.grant import Token
from quart import request, redirect, url_for, Response, session
from urllib.parse import urljoin

from oic.oic import Client
from oic.utils.authn.client import CLIENT_AUTHN_METHOD
from oic import rndstr
from oic.oic.message import AuthorizationResponse, RegistrationResponse
from oic.utils.keyio import KeyJar

from .. import app, store
from .web import get_key_for_user, JWT_ALGORITHM
from app.helpers.gitlab_user_utils import get_or_create_gitlab_user

logger = logging.getLogger(__name__)


class GitlabUserToken():

    def process(self, request, headers):

        m = re.search(r'bearer (?P<token>.+)', headers.get('Authorization', ''), re.IGNORECASE)
        if m:
            # logger.debug('Authorization header present, sudo token exchange')
            # logger.debug('outgoing headers: {}'.format(json.dumps(headers))
            access_token = m.group('token')
            decodentoken = jwt.decode(
                access_token, app.config['OIDC_PUBLIC_KEY'],
                algorithms=JWT_ALGORITHM,
                audience=app.config['OIDC_CLIENT_ID']
            )

            gl_token = store.get(get_key_for_user(decodentoken, 'gl_access_token'))
            headers['Authorization'] = "Bearer {}".format(gl_token.decode())
            headers['Renku-Token'] = access_token  # can be needed later in the request processing

        else:
            # logger.debug("No authorization header, returning empty auth headers")
            headers.pop('Sudo', None)

        return headers


SCOPE = ['openid', 'api', 'read_user', 'read_repository']

# We prepare the OIC client instance with the necessary configurations.
gitlab_client = Client(client_authn_method=CLIENT_AUTHN_METHOD)

try:
    gitlab_client.provider_config(
        issuer=app.config['GITLAB_URL'],
        keys=False,
    )

except:
    pass


# This fakes the response we would get from registering the client through the API
client_reg = RegistrationResponse(
    client_id=app.config['GITLAB_CLIENT_ID'],
    client_secret=app.config['GITLAB_CLIENT_SECRET'],
)
gitlab_client.store_registration_info(client_reg)

# gitlab /.well-known/openid-configuration doesn't take into account the protocol for generating its URLs
# so we have to manualy fix them here
gitlab_client.authorization_endpoint = "{}/oauth/authorize".format(app.config['GITLAB_URL'])
gitlab_client.token_endpoint = "{}/oauth/token".format(app.config['GITLAB_URL'])
gitlab_client.userinfo_endpoint = "{}/oauth/userinfo".format(app.config['GITLAB_URL'])
gitlab_client.jwks_uri = "{}/oauth/discovery/keys".format(app.config['GITLAB_URL'])
gitlab_client.keyjar = KeyJar()
gitlab_client.keyjar.load_keys({'jwks_uri': "{}/oauth/discovery/keys".format(app.config['GITLAB_URL'])}, app.config['GITLAB_URL'])


@app.route(urljoin(app.config['SERVICE_PREFIX'], 'auth/gitlab/login'))
def gitlab_login():

    state = rndstr()

    session['gitlab_state'] = state
    session['gitlab_ui_redirect_url'] = request.args.get('redirect_url')

    args = {
        'client_id': app.config['GITLAB_CLIENT_ID'],
        'response_type': 'code',
        'scope': SCOPE,
        'redirect_uri': app.config['HOST_NAME'] + url_for('gitlab_get_tokens'),
        'state': state
    }
    auth_req = gitlab_client.construct_AuthorizationRequest(request_args=args)
    login_url = auth_req.request(gitlab_client.authorization_endpoint)
    response = app.make_response(redirect(login_url))
    return response


@app.route(urljoin(app.config['SERVICE_PREFIX'], 'auth/gitlab/token'))
async def gitlab_get_tokens():

    # This is more about parsing the request data than any response data....
    authorization_parameters = gitlab_client.parse_response(
        AuthorizationResponse,
        info=request.query_string.decode('utf-8'),
        sformat='urlencoded'
    )

    if session['gitlab_state'] != authorization_parameters['state']:
        return 'Something went wrong while trying to log you in.'

    token_response = gitlab_client.do_access_token_request(
        scope=SCOPE,
        state=authorization_parameters['state'],
        request_args={
            'code': authorization_parameters['code'],
            'redirect_uri': app.config['HOST_NAME'] + url_for('gitlab_get_tokens'),
        }
    )

    a = jwt.decode(session['token'], verify=False)
    store.put(get_key_for_user(a, 'gl_access_token'), token_response['access_token'].encode())
    store.put(get_key_for_user(a, 'gl_refresh_token'), token_response['refresh_token'].encode())
    store.put(get_key_for_user(a, 'gl_id_token'), json.dumps(token_response['id_token'].to_dict()).encode())

    # chain logins to get the jupyterhub token
    response = await app.make_response(
        redirect(
            "{}?{}".format(
                url_for('jupyterhub_login'),
                urllib.parse.urlencode({'redirect_url': session['gitlab_ui_redirect_url']}),
            )
        )
    )

    return response


def get_gitlab_refresh_token(access_token):
    access_token = jwt.decode(
        access_token, app.config['OIDC_PUBLIC_KEY'],
        algorithms=JWT_ALGORITHM,
        audience=app.config['OIDC_CLIENT_ID']
    )
    to = Token(resp={'refresh_token': store.get(get_key_for_user(access_token, 'gl_refresh_token'))})
    refresh_token_response = gitlab_client.do_access_token_refresh(token=to)
    if 'access_token' in refresh_token_response:
        store.put(get_key_for_user(access_token, 'gl_access_token'), refresh_token_response['access_token'].encode())
        store.put(get_key_for_user(access_token, 'gl_refresh_token'), refresh_token_response['refresh_token'].encode())
    return refresh_token_response.get('access_token')


@app.route(urljoin(app.config['SERVICE_PREFIX'], 'auth/gitlab/logout'))
def gitlab_logout():
    logout_url = app.config['GITLAB_URL'] + '/users/sign_out'
    response = app.make_response(redirect(logout_url))

    return response
