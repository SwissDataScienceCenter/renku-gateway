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
"""Web auth routes."""

import jwt
import json
import time
import re
import logging
from flask import request, redirect, url_for, current_app

from oic.oic import Client
from oic.utils.authn.client import CLIENT_AUTHN_METHOD
from oic import rndstr
from oic.oic.message import AuthorizationResponse, RegistrationResponse
from oic.oauth2.grant import Token
from blinker import Namespace
from functools import wraps

from .. import app

# TODO: Currently, storing the short-lived login sessions inside a simple dictionary
# TODO: and using in-process signaling limits this service to be run by only one worker
# TODO: process and only one instance of it.

login_signals = Namespace()
login_done = login_signals.signal('login_done')

logger = logging.getLogger(__name__)
# Note that this part of the service should be seen as the server-side part of the UI or


JWT_SECRET = rndstr(size=32)
JWT_ALGORITHM = 'HS256'
SCOPE = ['openid']

# We need to specify that the cookie is valid for all .renga.build subdomains
if 'localhost' in app.config['HOST_NAME']:
    COOKIE_DOMAIN = None
else:
    COOKIE_DOMAIN = '.'.join([''] + app.config['HOST_NAME'].split('.')[1:])

# We use a short-lived dictionary to store ongoing login sessions.
# This should not grow in size and can easily be trashed when the service needs
# to be restarted.
login_sessions = {}


# We prepare the OIC client instance with the necessary configurations.
client = Client(client_authn_method=CLIENT_AUTHN_METHOD)

try:
    client.provider_config(
        issuer=app.config['OIDC_ISSUER']
    )
except:
    pass


# This fakes the response we would get from registering the client through the API
client_reg = RegistrationResponse(
    client_id=app.config['OIDC_CLIENT_ID'],
    client_secret=app.config['OIDC_CLIENT_SECRET']
)
client.store_registration_info(client_reg)


def swapped_token():
    def decorator(f):
        @wraps(f)
        def decorated_function(*args, **kwargs):
            headers = dict(request.headers)
            m = re.search(r'bearer (?P<token>.+)', headers.get('Authorization',''), re.IGNORECASE)

            if m and jwt.decode(m.group('token'), verify=False).get('typ') in ['Offline','Refresh']:
                logger.debug("Swapping the token")
                to = Token(resp={'refresh_token': m.group('token')})
                res = client.do_access_token_refresh(token=to)
                headers['Authorization'] = "Bearer {}".format(res.get('access_token'))

            return f(*args, **kwargs)
        return decorated_function
    return decorator


@app.route('/auth/login')
def login():

    session_id = rndstr(size=32)
    state = rndstr()
    login_session_token = jwt.encode({
        'id': session_id,
    }, JWT_SECRET, algorithm=JWT_ALGORITHM)

    login_sessions[session_id] = {
        'state': state,
        'ui_redirect_url': request.args.get('redirect_url'),
        'cli_token': request.args.get('cli_token'),
    }
    args = {
        'client_id': app.config['OIDC_CLIENT_ID'],
        'response_type': 'code',
        'scope': request.args.get('scope', SCOPE),
        'redirect_uri': app.config['HOST_NAME'] + url_for('get_tokens'),
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

    response = app.make_response(redirect(
        '/auth/info' if server_session.get('cli_token') else server_session['ui_redirect_url']
    ))
    response.set_cookie('access_token', value=token_response['access_token'], domain=COOKIE_DOMAIN)
    response.set_cookie('refresh_token', value=token_response['refresh_token'], domain=COOKIE_DOMAIN)
    response.set_cookie('id_token', value=json.dumps(token_response['id_token'].to_dict()), domain=COOKIE_DOMAIN)
    response.delete_cookie('session')

    if server_session.get('cli_token'):
        logger.debug("Notification for request {}".format(server_session.get('cli_token')))
        login_done.send(
            current_app._get_current_object(),
            cli_token=server_session.get('cli_token'),
            access_token=token_response['access_token'],
            refresh_token=token_response['refresh_token'],
        )

    return response


@app.route('/auth/info')
def info():

    t = request.args.get('cli_token')
    if t:
        signal = []

        def receive(sender, cli_token, access_token, refresh_token):
            if cli_token == t:
                signal.append((cli_token, access_token, refresh_token))
        login_done.connect(receive, current_app._get_current_object())
        timeout = 120
        logger.debug("Waiting for Keycloak callback for request {}".format(t))
        while not signal and timeout > 0:
            time.sleep(1)
            timeout -= 1
        login_done.disconnect(receive, current_app._get_current_object())
        if signal:
            return json.dumps({'access_token': signal[0][1], 'refresh_token': signal[0][2]})
        else:
            logger.debug("Timeout while waiting for request {}".format(t))
            return '{"error": "timeout"}'
    else:
        return "You can copy/paste the following tokens if needed and close this page: <br> Access Token: {}<br>Refresh Token: {}".format(
            request.cookies.get('access_token'), request.cookies.get('refresh_token'))


@app.route('/auth/logout')
def logout():
    logout_url = app.config['OIDC_ISSUER'] + '/protocol/openid-connect/logout?redirect_uri=' + \
                 request.args.get('redirect_url')
    response = app.make_response(redirect(logout_url))
    response.delete_cookie('access_token', domain=COOKIE_DOMAIN)
    response.delete_cookie('refresh_token', domain=COOKIE_DOMAIN)
    response.delete_cookie('id_token', domain=COOKIE_DOMAIN)

    return response
