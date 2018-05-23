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
"""Web auth routes."""

import jwt
import json

from flask import request, redirect, url_for

from oic.oic import Client
from oic.utils.authn.client import CLIENT_AUTHN_METHOD
from oic import rndstr
from oic.oic.message import AuthorizationResponse, RegistrationResponse


from .. import app
from app.settings import settings

# Note that this part of the service should be seen as the server-side part of the UI or

g = settings()

JWT_SECRET = rndstr(size=32)
JWT_ALGORITHM = 'HS256'
SCOPE = ['openid']

# We need to specify that the cookie is valid for all .renga.build subdomains
if 'localhost' in g['HOST_NAME']:
    COOKIE_DOMAIN = None
else:
    COOKIE_DOMAIN = '.'.join([''] + g['HOST_NAME'].split('.')[1:])

# We use a short-lived dictionary to store
# ongoing login sessions - should not grow in size and can easily be
# trashed. However, when running multiple instances of this the dictionary
# should be stored in a dedicated service.
login_sessions = {}


# We prepare the OIC client instance with the necessary configurations.
client = Client(client_authn_method=CLIENT_AUTHN_METHOD)
client.provider_config(
    issuer=g['OIDC_ISSUER']
)
# This fakes the response we would get from registering the client through the API
client_reg = RegistrationResponse(
    client_id=g['OIDC_CLIENT_ID'],
    client_secret=g['OIDC_CLIENT_SECRET']
)
client.store_registration_info(client_reg)


@app.route('/auth/login')
def login():

    session_id = rndstr(size=32)
    state = rndstr()
    login_session_token = jwt.encode({
        'id': session_id,
    }, JWT_SECRET, algorithm=JWT_ALGORITHM)

    login_sessions[session_id] = {
        'state': state,
        'ui_redirect_url': request.args.get('redirect_url')
    }
    args = {
        'client_id': g['OIDC_CLIENT_ID'],
        'response_type': 'code',
        'scope': SCOPE,
        'redirect_uri': g['HOST_NAME'] + url_for('get_tokens'),
        'state': state
    }
    auth_req = client.construct_AuthorizationRequest(request_args=args)
    login_url = auth_req.request(client.authorization_endpoint)
    response = app.make_response(redirect(login_url))
    response.set_cookie('session', value=login_session_token)
    return response


# TODO: Add token refresh method here
@app.route('/auth/token')
def get_tokens():

    browser_session = jwt.decode(request.cookies['session'], JWT_SECRET, algorithms=[JWT_ALGORITHM])
    server_session = login_sessions.pop(browser_session['id'], None)

    # This is more about parsing the request data than any response data....
    authorization_parameters = client.parse_response(
        AuthorizationResponse,
        info=request.query_string.decode('utf-8'),
        sformat='urlencoded'
    )

    if server_session['state'] != authorization_parameters['state']:
        return 'Something went wrong while trying to log you in.'

    token_response = client.do_access_token_request(
        scope=SCOPE,
        state=authorization_parameters['state'],
        request_args={
            'code': authorization_parameters['code']
        }
    )

    response = app.make_response(redirect(server_session['ui_redirect_url']))
    response.set_cookie('access_token', value=token_response['access_token'], domain=COOKIE_DOMAIN)
    response.set_cookie('refresh_token', value=token_response['refresh_token'], domain=COOKIE_DOMAIN)
    response.set_cookie('id_token', value=json.dumps(token_response['id_token'].to_dict()), domain=COOKIE_DOMAIN)
    response.delete_cookie('session')
    return response


@app.route('/auth/logout')
def logout():
    logout_url = g['OIDC_ISSUER'] + '/protocol/openid-connect/logout?redirect_uri=' + \
                 request.args.get('redirect_url')
    response = app.make_response(redirect(logout_url))
    response.delete_cookie('access_token', domain=COOKIE_DOMAIN)
    response.delete_cookie('refresh_token', domain=COOKIE_DOMAIN)
    response.delete_cookie('id_token', domain=COOKIE_DOMAIN)

    return response
