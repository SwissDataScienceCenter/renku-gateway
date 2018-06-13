

import pytest
from .. import app

import responses
import requests

from ..settings import settings
g = settings()


@pytest.fixture
def client():

    client = app.test_client()
    monkeypatch.set
    yield client

@responses.activate
def test_simple(client):

    test_url = g['GITLAB_URL'] + '/api/dummy'
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
    publickey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAjw2hj0o98jYxT0Z8hbd6INgJkVs2JvT6zXhHkN0UUWjGcRHF3e7Sc9GNI9wYljnBw47jqSmy2EftZ0UkGNjLENmGuLVC5r6vTneXfUht5t0+e5VelnM7yF7m9V3w/ms4wc0vDmSMa7pO5Vb+qjsTHgLTjQVqBIhGshxmZzKk6XDDVRlXe3SfLqwLX1biBmmvEOadU+2RhsHVW4rYMEaEHO1tCvRTsqiD7gVfk0XzZQg6KBEr2pDhz7hdWAfWK+/k1JiK/MNTs3FCfOqoBlTa6dZB/XRrKx0Pbi7y5Cr6BqqPUJzW5dbCYRzjmjL5CYx4KYHAjSSoCCpUjCHLILPPJQIDAQAB"

    keycloak_public_key = '-----BEGIN PUBLIC KEY-----\n' + publickey + '\n-----END PUBLIC KEY-----'

    path = 'api/v4/projects/'
    payload= {}
  #  faketoken = jwt.encode(payload=payload, key='fake-key', algorithm='RS256')

    git_url = g['GITLAB_URL'] + '/v4/projects/'
    responses.add(responses.GET, git_url, status=200)

    headers = {'Authorization':'Bearer eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJWQjhuYmZhZ2Jpbmo0TjNKS1BzdTE2dHlXdTcxcU5IaGVwLXducHRDeExzIn0.eyJqdGkiOiI5OTY1ODIzNC1kZjA0LTQxODAtYjQ5OC04YjFiNjA1YjZkN2UiLCJleHAiOjE1Mjg4OTY3NTcsIm5iZiI6MCwiaWF0IjoxNTI4ODk0OTU3LCJpc3MiOiJodHRwOi8va2V5Y2xvYWsucmVua3UuYnVpbGQ6ODA4MC9hdXRoL3JlYWxtcy9SZW5rdSIsImF1ZCI6ImdhdGV3YXkiLCJzdWIiOiI1ZGJkZWJhNy1lNDBmLTQyYTctYjQ2Yi02YjhhMDdjNjU5NjYiLCJ0eXAiOiJSZWZyZXNoIiwiYXpwIjoiZ2F0ZXdheSIsImF1dGhfdGltZSI6MCwic2Vzc2lvbl9zdGF0ZSI6Ijg5OWJmZTNjLTVhN2UtNGVhMC1iMzQwLWI0MTc5YjI5Nzk2OCIsInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJ1bWFfYXV0aG9yaXphdGlvbiJdfSwicmVzb3VyY2VfYWNjZXNzIjp7ImFjY291bnQiOnsicm9sZXMiOlsibWFuYWdlLWFjY291bnQiLCJtYW5hZ2UtYWNjb3VudC1saW5rcyIsInZpZXctcHJvZmlsZSJdfX19.SQCHTMae-ZZXSxhqxh4YrzSshpvgD_HfQUEymbZ_SWByMQ2rB94JHM0oD9-F14MtiY-sJ-kiK6fmkReUr2LiVnmuY9FH1I1yTRj26VEAiV3lPsIz7hy-I583PPOdqYncHABZmW4XtbD4cIidBN56K2h0gDb782hbkdA7XA1QIopJWQ9I5r8PC96JgYukw1PoYKA-Br3ewZn-IjrNs71b1NraX9U1D_X9-mdtCOowehNU_Hzs9cMVqkHVrIiAThCHpJ4aDHnUCbnBwElrC9bLbwksIwRJzOVCDiK0CW1Woe6uG4gEhq3bLW_zWw3czkTCnjDzg8YuuV-DjidzZ6ORNQ', 'Private-Token': 'dummy-secret', 'Sudo': 'demo'}

    rv = client.get(path, headers=headers)

    assert rv.status_code == 200
  #  assert b'No authorization header found' not in rv.data
