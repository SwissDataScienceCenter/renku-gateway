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
import logging
import re
import urllib.parse
from oic.oauth2.grant import Token
from quart import request, redirect, url_for, current_app, Response, session, render_template
from urllib.parse import urljoin, quote_plus

from oic.oic import Client
from oic.utils.authn.client import CLIENT_AUTHN_METHOD
from oic import rndstr
from oic.oic.message import AuthorizationResponse, RegistrationResponse

from .. import app, store


logger = logging.getLogger(__name__)
# Note that this part of the service should be seen as the server-side part of the UI or

JWT_ALGORITHM = 'RS256'
SCOPE = ['openid']

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


def get_valid_token(headers):
    """
    Look for a fresh and valid token, first in headers, then in the session.

    If a refresh token is available, it can be swapped for an access token.
    """
    m = re.search(r'bearer (?P<token>.+)', headers.get('Authorization', ''), re.IGNORECASE)

    if m:
        if jwt.decode(m.group('token'), verify=False).get('typ') in ['Offline', 'Refresh']:
            logger.debug("Swapping the token")
            to = Token(resp={'refresh_token': m.group('token')})
            token_response = client.do_access_token_refresh(token=to)

            if 'access_token' in token_response:
                try:
                    a = jwt.decode(
                        token_response['access_token'], app.config['OIDC_PUBLIC_KEY'],
                        algorithms=JWT_ALGORITHM,
                        audience=app.config['OIDC_CLIENT_ID']
                    )
                    return token_response
                except:
                    return None
        else:
            try:
                jwt.decode(
                    m.group('token'),
                    app.config['OIDC_PUBLIC_KEY'],
                    algorithms=JWT_ALGORITHM,
                    audience=app.config['OIDC_CLIENT_ID']
                )

                return {'access_token': m.group('token')}

            except:
                return None
    else:
        try:
            if headers.get('X-Requested-With') == 'XMLHttpRequest' and 'token' in session:
                a = jwt.decode(
                    session.get('token'),
                    app.config['OIDC_PUBLIC_KEY'],
                    algorithms=JWT_ALGORITHM,
                    audience=app.config['OIDC_CLIENT_ID']
                )
                access_token = store.get(get_key_for_user(a, 'kc_access_token')).decode()
                jwt.decode(
                    access_token,
                    app.config['OIDC_PUBLIC_KEY'],
                    algorithms=JWT_ALGORITHM,
                    audience=app.config['OIDC_CLIENT_ID']
                )
                return {'access_token': access_token}

        except:
            if 'token' in session and jwt.decode(session.get('token'), verify=False).get('typ') in ['Offline', 'Refresh']:
                logger.debug("Swapping the token")
                to = Token(resp={'refresh_token': session.get('token')})

                token_response = client.do_access_token_refresh(token=to)

                if 'access_token' in token_response:
                    try:
                        a = jwt.decode(
                            token_response['access_token'], app.config['OIDC_PUBLIC_KEY'],
                            algorithms=JWT_ALGORITHM,
                            audience=app.config['OIDC_CLIENT_ID']
                        )
                        # session['token'] = token_response['refresh_token']  # uncomment to allow sessions to be extended
                        store.put(get_key_for_user(a, 'kc_access_token'), token_response['access_token'].encode())
                        store.put(get_key_for_user(a, 'kc_refresh_token'), token_response['refresh_token'].encode())
                        store.put(get_key_for_user(a, 'kc_id_token'), json.dumps(token_response['id_token'].to_dict()).encode())
                        return token_response
                    except:
                        return None

    return None


def get_key_for_user(token, name):
    return "cache_{}_{}".format(token.get('sub'), name)

LOGIN_SEQUENCE = ['gitlab_login', 'jupyterhub_login']

@app.route(urljoin(app.config['SERVICE_PREFIX'], 'auth/login/next'))
async def login_next():

    if session['login_seq'] < len(LOGIN_SEQUENCE):
        return await render_template('redirect.html', redirect_url=url_for(LOGIN_SEQUENCE[session['login_seq']]))
    else:
        return redirect(session['ui_redirect_url'])


@app.route(urljoin(app.config['SERVICE_PREFIX'], 'auth/login'))
async def login():

    state = rndstr()

    session['state'] = state
    session['login_seq'] = 0
    session['ui_redirect_url'] = request.args.get('redirect_url')
    session['cli_token'] = request.args.get('cli_token')
    if session['cli_token']:
        session['ui_redirect_url'] = app.config['HOST_NAME'] + url_for('info')

    args = {
        'client_id': app.config['OIDC_CLIENT_ID'],
        'response_type': 'code',
        'scope': SCOPE,
        'redirect_uri': app.config['HOST_NAME'] + url_for('get_tokens'),
        'state': state
    }
    auth_req = client.construct_AuthorizationRequest(request_args=args)
    login_url = auth_req.request(client.authorization_endpoint)
    response = await app.make_response(redirect(login_url))

    return response


@app.route(urljoin(app.config['SERVICE_PREFIX'], 'auth/token'))
async def get_tokens():

    # This is more about parsing the request data than any response data....
    authorization_parameters = client.parse_response(
        AuthorizationResponse,
        info=request.query_string.decode('utf-8'),
        sformat='urlencoded'
    )

    if session.get('state') != authorization_parameters['state']:
        return 'Something went wrong while trying to log you in.'

    token_response = client.do_access_token_request(
        scope=SCOPE,
        state=authorization_parameters['state'],
        request_args={
            'code': authorization_parameters['code'],
            'redirect_uri': app.config['HOST_NAME'] + url_for('get_tokens'),
        }
    )

    # chain logins
    response = await app.make_response(redirect(url_for('login_next')))


    a = jwt.decode(
        token_response['refresh_token'], app.config['OIDC_PUBLIC_KEY'],
        algorithms=JWT_ALGORITHM,
        audience=app.config['OIDC_CLIENT_ID']
    )
    session['token'] = token_response['refresh_token']
    store.put(get_key_for_user(a, 'kc_access_token'), token_response['access_token'].encode())
    store.put(get_key_for_user(a, 'kc_refresh_token'), token_response['refresh_token'].encode())
    store.put(get_key_for_user(a, 'kc_id_token'), json.dumps(token_response['id_token'].to_dict()).encode())

    # we can already tell the CLI which token to use
    if session.get('cli_token'):
        logger.debug("Notification for request {}".format(session.get('cli_token')))

        key = "cli_{}".format(hashlib.sha256(session.get('cli_token').encode()).hexdigest())
        store.put(key, json.dumps({'access_token': token_response['access_token'], 'refresh_token': token_response['refresh_token']}).encode())

    return response


@app.route(urljoin(app.config['SERVICE_PREFIX'], 'auth/info'))
async def info():

    t = request.args.get('cli_token')
    if t:
        timeout = 120
        key = "cli_{}".format(hashlib.sha256(t.encode()).hexdigest())
        logger.debug("Waiting for Keycloak callback for request {}".format(t))
        val = store.get(key)
        while not val and timeout > 0:
            time.sleep(3)
            timeout -= 3
            val = store.get(key)
        if val:
            store.delete(key)
            return val
        else:
            logger.debug("Timeout while waiting for request {}".format(t))
            return '{"error": "timeout"}'
    else:

        if 'token' not in session:
            return await app.make_response(redirect("{}?redirect_url={}".format(url_for('login'), quote_plus(url_for('info')))))

        try:
            a = jwt.decode(
                session['token'],
                app.config['OIDC_PUBLIC_KEY'],
                algorithms=JWT_ALGORITHM,
                audience=app.config['OIDC_CLIENT_ID']
            )  # TODO: logout and redirect if fails because of expired

            return "You can copy/paste the following tokens if needed and close this page: <br> Access Token: {}<br>Refresh Token: {}".format(
                store.get(get_key_for_user(a, 'kc_access_token')).decode(), store.get(get_key_for_user(a, 'kc_refresh_token')).decode())

        except jwt.ExpiredSignatureError:
            return await app.make_response(redirect("{}?redirect_url={}".format(url_for('login'), quote_plus(url_for('info')))))


@app.route(urljoin(app.config['SERVICE_PREFIX'], 'auth/user'))
async def user():

    if 'token' not in session:
        return await app.make_response(redirect("{}?redirect_url={}".format(url_for('login'), quote_plus(url_for('user')))))
    try:
        a = jwt.decode(
            session['token'],
            app.config['OIDC_PUBLIC_KEY'],
            algorithms=JWT_ALGORITHM,
            audience=app.config['OIDC_CLIENT_ID']
        )  # TODO: logout and redirect if fails because of expired

        return store.get(get_key_for_user(a, 'kc_id_token')).decode()

    except jwt.ExpiredSignatureError:
            return await app.make_response(redirect("{}?redirect_url={}".format(url_for('login'), quote_plus(url_for('user')))))


@app.route(urljoin(app.config['SERVICE_PREFIX'], 'auth/logout'))
async def logout():

    logout_url = '{}/protocol/openid-connect/logout?{}'.format(
        app.config['OIDC_ISSUER'],
        urllib.parse.urlencode({'redirect_uri': request.args.get('redirect_url')}),
    )

    if 'token' in session:
        a = jwt.decode(session['token'], verify=False)

        # cleanup the session in redis immediately
        cookie_val = request.cookies.get('session').split(".")[0]
        store.delete(cookie_val)
        session.clear()

        for k in store.keys(prefix=get_key_for_user(a, '')):
            store.delete(k)

    return await app.make_response(redirect(logout_url))
