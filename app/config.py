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
"""Global settings."""

import json
import os
import re
import requests
import sys
from time import sleep
from logging import getLogger
from collections import OrderedDict

logger = getLogger(__name__)


config = dict()

config['HOST_NAME'] = os.environ.get('HOST_NAME', 'http://gateway.renku.build')

config['SECRET_KEY'] = os.environ.get('GATEWAY_SECRET_KEY', 'dummy-secret')

# We need to specify that the cookie is valid for all .renku.build subdomains
if 'gateway.renku.build' in config['HOST_NAME']:
    config['SESSION_COOKIE_DOMAIN'] = '.'.join([''] + config['HOST_NAME'].split('.')[1:])
else:
    config['SESSION_COOKIE_DOMAIN'] = None

config['SESSION_COOKIE_HTTPONLY'] = True
config['SESSION_COOKIE_SECURE'] = config['HOST_NAME'].startswith('https')

config['REDIS_HOST'] = os.environ.get('GATEWAY_REDIS_HOST', 'renku-gw-redis')

config['RENKU_ENDPOINT'] = os.environ.get('RENKU_ENDPOINT', 'http://renku.build')
config['GITLAB_URL'] = os.environ.get('GITLAB_URL', 'http://gitlab.renku.build')
config['GITLAB_PASS'] = os.environ.get('GITLAB_PASS', 'dummy-secret')
config['GITLAB_CLIENT_ID'] = os.environ.get('GITLAB_CLIENT_ID', 'renku-ui')
config['GITLAB_CLIENT_SECRET'] = os.environ.get('GITLAB_CLIENT_SECRET', 'no-secret-needed')

config['OIDC_ISSUER'] = os.environ.get('KEYCLOAK_URL', 'http://keycloak.renku.build:8080') \
                        + '/auth/realms/Renku'
config['OIDC_CLIENT_ID'] = os.environ.get('OIDC_CLIENT_ID', 'gateway')
config['OIDC_CLIENT_SECRET'] = os.environ.get('OIDC_CLIENT_SECRET', 'dummy-secret')
config['SERVICE_PREFIX'] = os.environ.get('GATEWAY_SERVICE_PREFIX', '/')

# Get the public key of the OIDC provider to verify access- and refresh_tokens
# TODO: The public key of the OIDC provider should go to the app context and be refreshed
# TODO: regularly or whenever the validation of a token fails and the public key has not been
# TODO: updated in a while.


config['GATEWAY_ENDPOINT_CONFIG_FILE'] = os.environ.get('GATEWAY_ENDPOINT_CONFIG_FILE', 'endpoints.json')


def load_config():
    from . import app
    app.config['GATEWAY_ENDPOINT_CONFIG'] = {}
    try:
        with open(app.config['GATEWAY_ENDPOINT_CONFIG_FILE']) as f:
            c = json.load(f, object_pairs_hook=OrderedDict)
        for k, v in c.items():
            app.config['GATEWAY_ENDPOINT_CONFIG'][re.compile(r"{}(?P<remaining>.*)".format(k))] = v
    except:
        logger.error("Error reading endpoints config file", exc_info=True)

    logger.debug(app.config['GATEWAY_ENDPOINT_CONFIG'])


if "pytest" in sys.modules:
    okKey = True
else:
    okKey = False
attempts = 0

while attempts < 20 and not okKey:
    attempts += 1
    try:
        raw_key = requests.get(config['OIDC_ISSUER']).json()['public_key']
        config['OIDC_PUBLIC_KEY'] = '-----BEGIN PUBLIC KEY-----\n{}\n-----END PUBLIC KEY-----'.format(raw_key)
        okKey = True
        logger.info('Obtained public key from Keycloak.')
    except:
        logger.info('Could not get public key from Keycloak, trying again...')
        sleep(10)


if not okKey:
    logger.info('Could not get public key from Keycloak, giving up.')
    exit(1)
