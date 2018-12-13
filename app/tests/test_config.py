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

import pytest
import responses
from quart import Response

from app.config import load_config
from app.processors.base_processor import BaseProcessor

from .. import app
from .test_data import PUBLIC_KEY


@pytest.fixture
def client():
    app.config['TESTING'] = True
    app.config['OIDC_PUBLIC_KEY'] = PUBLIC_KEY
    app.config['GATEWAY_ENDPOINT_CONFIG_FILE'] = 'test.json'
    load_config()
    client = app.test_client()
    yield client


def aiotest(func):
    @functools.wraps(func)
    def _func(*args, **kwargs):
        loop = asyncio.new_event_loop()
        loop.run_until_complete(func(*args, **kwargs))
        loop.close()

    return _func


@aiotest
async def test_empty_config(client):

    app.config['GATEWAY_ENDPOINT_CONFIG'] = {}
    rv = await client.get(urljoin(app.config['SERVICE_PREFIX'], 'something'))

    assert rv.status_code == 404
    assert json.loads(await rv.get_data()) == {
        "error": "No processor found for this path"
    }


@aiotest
async def test_catch_all(client):
    with open(app.config['GATEWAY_ENDPOINT_CONFIG_FILE'], 'w') as f:
        json.dump({
            "": {
                "endpoint": "http://localhost/api",
                "processor": "app.tests.test_config.DummyProcessor",
                "auth": "app.tests.test_config.DummyAuth"
            }
        }, f)
    load_config()

    rv = await client.get(
        urljoin(
            app.config['SERVICE_PREFIX'], 'something/interesting?p=nothing'
        )
    )

    assert rv.status_code == 200
    assert (await rv.get_data()) == b'something/interesting'
    assert rv.headers.get('X-DummyAuth') == 'ok'


@aiotest
async def test_regex_config(client):
    with open(app.config['GATEWAY_ENDPOINT_CONFIG_FILE'], 'w') as f:
        json.dump({
            "obj(ect)?/[\d-]+/": {
                "endpoint": "http://localhost/api",
                "processor": "app.tests.test_config.DummyProcessor"
            }
        }, f)
    load_config()

    rv = await client.get(
        urljoin(app.config['SERVICE_PREFIX'], 'obj/23-3/issues?p=nothing')
    )

    assert rv.status_code == 200
    assert (await rv.get_data()) == b'issues'

    rv = await client.get(
        urljoin(app.config['SERVICE_PREFIX'], 'object/-/issues/4?p=nothing')
    )

    assert rv.status_code == 200
    assert (await rv.get_data()) == b'issues/4'

    rv = await client.get(
        urljoin(app.config['SERVICE_PREFIX'], 'objects/-/issues/4?p=nothing')
    )

    assert rv.status_code == 404
    assert json.loads(await rv.get_data()) == {
        "error": "No processor found for this path"
    }

    rv = await client.get(
        urljoin(app.config['SERVICE_PREFIX'], 'obj/2a/issues/4?p=nothing')
    )

    assert rv.status_code == 404
    assert json.loads(await rv.get_data()) == {
        "error": "No processor found for this path"
    }


class DummyProcessor(BaseProcessor):
    async def process(self, request, header):
        rsp = Response(self.path)
        rsp.headers = header
        return rsp


class DummyAuth():
    def process(self, request, header):
        header['X-DummyAuth'] = 'ok'
        return header
