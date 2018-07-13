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

import os
import requests


config = dict()
config['HOST_NAME'] = os.environ.get('HOST_NAME', 'http://gateway.renku.build')
config['RENKU_ENDPOINT'] = os.environ.get('RENKU_ENDPOINT', 'http://renku.build')
config['GITLAB_URL'] = os.environ.get('GITLAB_URL', 'http://gitlab.renku.build')
config['GITLAB_PASS'] = os.environ.get('GITLAB_PASS', 'dummy-secret')

config['OIDC_ISSUER'] = os.environ.get('KEYCLOAK_URL', 'http://keycloak.renku.build:8080') \
                        + '/auth/realms/Renku'
config['OIDC_CLIENT_ID'] = os.environ.get('OIDC_CLIENT_ID', 'gateway')
config['OIDC_CLIENT_SECRET'] = os.environ.get('OIDC_CLIENT_SECRET', 'dummy-secret')
config['SERVICE_PREFIX'] = os.environ.get('GATEWAY_SERVICE_PREFIX', '/')

# Get the public key of the OIDC provider to verify access- and refresh_tokens
# TODO: The public key of the OIDC provider should go to the app context and be refreshed
# TODO: regularly or whenever the validation of a token fails and the public key has not been
# TODO: updated in a while.
try:
    raw_key = requests.get(config['OIDC_ISSUER']).json()['public_key']
    config['OIDC_PUBLIC_KEY'] = '-----BEGIN PUBLIC KEY-----\n{}\n-----END PUBLIC KEY-----'.format(raw_key)
except:
    config['OIDC_PUBLIC_KEY'] = None

