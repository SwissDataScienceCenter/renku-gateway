

import os

import pytest
from .. import app
import flask

import responses
import requests

from ..settings import settings
g = settings()

@pytest.fixture
def client():
  #  db_fd, flask.app.config['DATABASE'] = tempfile.mkstemp()
    #flask.app.config['TESTING'] = True
    client = app.test_client()


    yield client

@responses.activate
def test_simple(client):
    test_url = g['GITLAB_URL'] + '/api/dummy'
    responses.add(responses.GET, test_url,
                  json={'error': 'not found'}, status=404)

    rv = client.get('/api/dummy')
    print(rv)
    resp = requests.get(test_url)

    assert resp.json() == {"error": "not found"}

    assert len(responses.calls) == 1
    assert responses.calls[0].request.url == test_url
    assert responses.calls[0].response.text == '{"error": "not found"}'


def test_empty_db(client):
    """Start with a blank database."""

    rv = client.get('/api/dummy')
    assert b'Dummy works' in rv.data