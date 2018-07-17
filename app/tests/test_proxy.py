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
""" Test for the proxy """


import pytest
from .. import app
import responses
import requests
import jwt
from .test_data import PUBLIC_KEY, PRIVATE_KEY, TOKEN_PAYLOAD

@pytest.fixture
def client():
    app.config['TESTING'] = True
    app.config['OIDC_PUBLIC_KEY'] = PUBLIC_KEY
    client = app.test_client()
    yield client

@responses.activate
def test_simple(client):

    test_url = app.config['GITLAB_URL'] + '/dummy'
    responses.add(responses.GET, test_url,
                  json={'error': 'not found'}, status=404)

    rv = client.get('/dummy')
    resp = requests.get(test_url)

    assert resp.json() == {"error": "not found"}

    assert len(responses.calls) == 1
    assert responses.calls[0].request.url == test_url
    assert responses.calls[0].response.text == '{"error": "not found"}'


def test_empty_db(client):
    """Start with a blank database."""

    rv = client.get('/dummy')
    assert b'Dummy works' in rv.data

@responses.activate
def test_passthrough_nopubkeyflow(client):
    # If no keycloak token exists, the pass through should fail with 500
    app.config['OIDC_PUBLIC_KEY'] = None
    path = '/api/gitlab/v4/projects/'
    rv = client.get(path)
    assert rv.status_code == 500
    assert b'"Ooops, something went wrong internally' in rv.data

@responses.activate
def test_passthrough_happyflow(client):
    # If a request does has the required headers, it should be able to pass through
    access_token = jwt.encode(payload=TOKEN_PAYLOAD, key=PRIVATE_KEY, algorithm='RS256').decode('utf-8')
    headers = {'Authorization': 'Bearer {}'.format(access_token)}
    path = '/api/v4/projects/'

    gitlab_endpoint_url = app.config['GITLAB_URL'] + path
    responses.add(responses.GET, gitlab_endpoint_url, status=200)

    rv = client.get('/v4/projects/', headers=headers)

    assert rv.status_code == 200
    assert b'No authorization header found' not in rv.data
