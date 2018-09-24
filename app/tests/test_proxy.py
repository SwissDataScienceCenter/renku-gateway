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
import asyncio
import functools
from .. import app
import responses
import requests
import jwt
import json
from urllib.parse import urljoin
from .test_data import PUBLIC_KEY, PRIVATE_KEY, TOKEN_PAYLOAD, GITLAB_PROJECTS, GITLAB_ISSUES, GATEWAY_PROJECT
from app.config import load_config


@pytest.fixture
def client():
    app.config['TESTING'] = True
    app.config['OIDC_PUBLIC_KEY'] = PUBLIC_KEY
    app.config['GATEWAY_ENDPOINT_CONFIG_FILE'] = 'endpoints.json'
    load_config()
    client = app.test_client()
    yield client


def aiotest(func):
    @functools.wraps(func)
    def _func(*args, **kwargs):
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(None)
        loop.run_until_complete(func(*args, **kwargs))
        loop.close()
    return _func


@responses.activate
def test_simple(client):

    test_url = app.config['GITLAB_URL'] + '/dummy'
    responses.add(responses.GET, test_url,
                  json={'error': 'not found'}, status=404)

    resp = requests.get(test_url)

    assert resp.json() == {"error": "not found"}

    assert len(responses.calls) == 1
    assert responses.calls[0].request.url == test_url
    assert responses.calls[0].response.text == '{"error": "not found"}'


@aiotest
async def test_empty_db(client):
    """Start with a blank database."""

    rv = await client.get('/dummy')
    assert b'Dummy works' in (await rv.get_data())


@aiotest
async def test_passthrough_nopubkeyflow(client):
    # If no keycloak token exists, the pass through should fail with 500
    app.config['OIDC_PUBLIC_KEY'] = None
    path = urljoin(app.config['SERVICE_PREFIX'], 'v4/projects/')
    rv = await client.get(path)
    assert rv.status_code == 500
    assert b'"Ooops, something went wrong internally' in (await rv.get_data())


## TODO: currently no endpoint absolutely requires a token
# @responses.activate
# def test_passthrough_notokenflow(client):
#    # If a request does not have the required header it should not be let through
#    path = urljoin(app.config['SERVICE_PREFIX'], 'v4/projects/')
#    rv = client.get(path)
#    assert rv.status_code == 401
#    assert b'No authorization header found' in (await rv.get_data())


## TODO: currently the project mapper is not used, but we keep the other response for future use.

@responses.activate
@aiotest
async def test_gitlab_happyflow(client):
    # If a request does has the required headers, it should be able to pass through
    access_token = jwt.encode(payload=TOKEN_PAYLOAD, key=PRIVATE_KEY, algorithm='RS256').decode('utf-8')
    headers = {'Authorization': 'Bearer {}'.format(access_token)}

    from .. import store
    store.put('cache_5dbdeba7-e40f-42a7-b46b-6b8a07c65966_gl_access_token', 'some_token'.encode())

    responses.add(responses.GET, app.config['GITLAB_URL'] + '/api/v4/projects', json=GITLAB_PROJECTS, status=200)
    responses.add(responses.GET, app.config['GITLAB_URL'] + "/api/v4/projects/1/repository/files/README.md/raw?ref=master", body="test", status=200)
    responses.add(responses.GET, app.config['GITLAB_URL'] + "/api/v4/projects/1/issues?scope=all", json=GITLAB_ISSUES, status=200)
    responses.add(responses.GET, app.config['GITLAB_URL'] + "/api/v4/projects/1/issues/1/award_emoji", json=[], status=200)
    responses.add(responses.GET, app.config['GITLAB_URL'] + "/api/v4/projects/1/issues/1/notes", json=[], status=200)
    responses.add(responses.GET, app.config['GITLAB_URL'] + '/api/v4/users', json=[{'username': 'foo'}])

    rv = await client.get(urljoin(app.config['SERVICE_PREFIX'], 'v4/projects'), headers=headers)

    assert rv.status_code == 200
    assert json.loads(await rv.get_data()) == GITLAB_PROJECTS


@responses.activate
@aiotest
async def test_service_happyflow(client):
    # If a request does has the required headers, it should be able to pass through
    access_token = jwt.encode(payload=TOKEN_PAYLOAD, key=PRIVATE_KEY, algorithm='RS256').decode('utf-8')
    headers = {'Authorization': 'Bearer {}'.format(access_token)}

    responses.add(responses.POST, app.config['RENKU_ENDPOINT'] + '/service/storage/object/23/meta', json={'id': 1}, status=201)

    rv = await client.post(urljoin(app.config['SERVICE_PREFIX'], 'objects/23/meta'), headers=headers)

    assert rv.status_code == 201
    assert json.loads(await rv.get_data()) == {'id': 1}
