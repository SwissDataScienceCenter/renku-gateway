

import pytest
from .. import app
import jwt

import responses
import requests

from ..settings import settings
g = settings()

@pytest.fixture
def client():
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

@responses.activate
def test_passthrough_notokenflow(client):
    # If a request does not have the required header it should not be let through

    path = '/api/v4/projects/'
    rv = client.get(path)
    assert rv.status_code == 401
    assert b'No authorization header found' in rv.data

@responses.activate
# Test won't work
# Need a way to get a fake keycloak public key
def test_passthrough_happyflow(client):
    # If a request does has the required headers, it should be able to pass through

    path = 'api/v4/projects/'

  #  payload= {}
  #  faketoken = jwt.encode(payload=payload, key='fake-key', algorithm='RS256')

    git_url = g['GITLAB_URL'] + '/v4/projects/'
    responses.add(responses.GET, git_url, status=200)

    headers = {'Authorization':'Bearer ey', 'Private-Token': 'dummy-secret', 'Sudo': 'demo'}

    rv = client.get(path, headers=headers)

   # assert rv.status_code == 200
   # assert b'No authorization header found' not in rv.data
