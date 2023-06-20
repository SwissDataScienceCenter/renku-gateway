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

import jwt
import pytest
import requests
import responses

from .. import app
from ..auth.oauth_client import RenkuWebApplicationClient
from ..auth.oauth_provider_app import OAuthProviderApp
from ..auth.oauth_redis import OAuthRedis
from ..auth.utils import get_redis_key_from_token
from .test_data import (
    PRIVATE_KEY,
    PROVIDER_APP_DICT,
    PUBLIC_KEY,
    SECRET_KEY,
    TOKEN_PAYLOAD,
    REDIS_PASSWORD,
)

# TODO: Completely refactor all tests, massively improve test coverage.
# TODO: https://github.com/swissdatasciencecenter/renku-gateway/issues/92


@pytest.fixture
def client():
    app.app_context().push()
    app.config["TESTING"] = True
    app.config["OIDC_PUBLIC_KEY"] = PUBLIC_KEY
    app.config["SECRET_KEY"] = SECRET_KEY
    app.config["REDIS_PASSWORD"] = REDIS_PASSWORD
    client = app.test_client()
    yield client


@responses.activate
def test_simple(client):
    test_url = app.config["GITLAB_URL"] + "/dummy"
    responses.add(responses.GET, test_url, json={"error": "not found"}, status=404)

    resp = requests.get(test_url)

    assert resp.json() == {"error": "not found"}

    assert len(responses.calls) == 1
    assert responses.calls[0].request.url == test_url
    assert responses.calls[0].response.text == '{"error": "not found"}'


def test_health_endpoint(client):
    rv = client.get("/health")
    assert b'"Up and running"' in (rv.get_data())


# TODO: currently no endpoint absolutely requires a token
# @responses.activate
# def test_passthrough_notokenflow(client):
#    # If a request does not have the required header it should not be let through
#    path = urljoin(app.config['SERVICE_PREFIX'], 'v4/projects/')
#    rv = client.get(path)
#    assert rv.status_code == 401
#    assert b'No authorization header found' in (rv.get_data())

# TODO: currently the project mapper is not used, but we keep the other response
# TODO: for future use.


def test_gitlab_happyflow(client):
    """If a request has the required headers, it should be able to pass through."""

    def set_dummy_oauth_client(token, key_suffix):
        provider_app = OAuthProviderApp(**PROVIDER_APP_DICT)
        redis_key = get_redis_key_from_token(access_token, key_suffix=key_suffix)
        oauth_client = RenkuWebApplicationClient(
            access_token=token, provider_app=provider_app
        )
        app.store.set_oauth_client(redis_key, oauth_client)

    access_token = jwt.encode(payload=TOKEN_PAYLOAD, key=PRIVATE_KEY, algorithm="RS256")
    headers = {"Authorization": "Bearer {}".format(access_token)}

    app.store = OAuthRedis(hex_key=app.config["SECRET_KEY"])

    set_dummy_oauth_client(access_token, app.config["CLI_SUFFIX"])

    set_dummy_oauth_client("some_token", app.config["GL_SUFFIX"])

    rv = client.get("/?auth=gitlab", headers=headers)

    assert rv.status_code == 200
    assert "Bearer some_token" == rv.headers["Authorization"]
