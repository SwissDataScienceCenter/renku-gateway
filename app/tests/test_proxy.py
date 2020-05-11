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

import asyncio
import functools
import json
from urllib.parse import urljoin

import jwt
import pytest
import requests
import responses

from aioresponses import aioresponses

from .. import app
from .test_data import (
    GITLAB_ISSUES,
    GITLAB_PROJECTS,
    PRIVATE_KEY,
    PUBLIC_KEY,
    TOKEN_PAYLOAD,
)


@pytest.fixture
def client():
    app.config["TESTING"] = True
    app.config["OIDC_PUBLIC_KEY"] = PUBLIC_KEY
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

    test_url = app.config["GITLAB_URL"] + "/dummy"
    responses.add(responses.GET, test_url, json={"error": "not found"}, status=404)

    resp = requests.get(test_url)

    assert resp.json() == {"error": "not found"}

    assert len(responses.calls) == 1
    assert responses.calls[0].request.url == test_url
    assert responses.calls[0].response.text == '{"error": "not found"}'


@aiotest
async def test_health_endpoint(client):
    rv = await client.get("/health")
    assert b'"Up and running"' in (await rv.get_data())


## TODO: currently no endpoint absolutely requires a token
# @responses.activate
# def test_passthrough_notokenflow(client):
#    # If a request does not have the required header it should not be let through
#    path = urljoin(app.config['SERVICE_PREFIX'], 'v4/projects/')
#    rv = client.get(path)
#    assert rv.status_code == 401
#    assert b'No authorization header found' in (await rv.get_data())

## TODO: currently the project mapper is not used, but we keep the other response for future use.


@aiotest
async def test_gitlab_happyflow(client):
    # If a request does has the required headers, it should be able to pass through
    access_token = jwt.encode(
        payload=TOKEN_PAYLOAD, key=PRIVATE_KEY, algorithm="RS256"
    ).decode("utf-8")
    headers = {"Authorization": "Bearer {}".format(access_token)}

    from .. import store
    from base64 import b64encode

    store.put(
        b64encode(
            "cache_5dbdeba7-e40f-42a7-b46b-6b8a07c65966_gl_access_token".encode()
        ).decode("utf-8"),
        "some_token".encode(),
    )

    rv = await client.get("/?auth=gitlab", headers=headers)

    assert rv.status_code == 200
    assert "Bearer some_token" == rv.headers["Authorization"]
