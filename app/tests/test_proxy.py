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

    test_url = app.config['GITLAB_URL'] + '/api/dummy'
    responses.add(responses.GET, test_url,
                  json={'error': 'not found'}, status=404)

    rv = client.get('/api/dummy')
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
def test_passthrough_nopubkeyflow(client):
    # If no keycloak token exists, the pass through should fail with 500

    path = '/api/v4/projects/'
    rv = client.get(path)
    assert rv.status_code == 500
    assert b'Keycloak public key not defined' in rv.data

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
    access_token = jwt.encode(payload=TOKEN_PAYLOAD, key=PRIVATE_KEY, algorithm='RS256').decode('utf-8')
    headers = {'Authorization': 'Bearer {}'.format(access_token)}
    path = '/api/v4/projects/'

    gitlab_endpoint_url = app.config['GITLAB_URL'] + path
    responses.add(responses.GET, gitlab_endpoint_url, status=200)

    rv = client.get('api/v4/projects/' , headers=headers)

    assert rv.status_code == 200
#  assert b'No authorization header found' not in rv.data