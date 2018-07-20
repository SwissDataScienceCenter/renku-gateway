from app.processors.base_processor import BaseProcessor
from .. import app
from app.helpers.gitlab_parsers import parse_project

import logging
import json
import requests

from flask import Response

import jwt
from functools import wraps

logger = logging.getLogger(__name__)


# TODO use token
# def with_tokens(f):
#      """Function decorator to ensure we have OIDC tokens"""
#      return 0
def authorize():
    def decorator(f):
        @wraps(f)
        def decorated_function(self, request, headers):
            if 'Authorization' in headers:
                logger.debug("Authorization header present, sudo token exchange")
                access_token = headers.get('Authorization')[7:]
                del headers['Authorization']
                headers['Private-Token'] = app.config['GITLAB_PASS']

                # Decode token to get user id
                decodentoken = jwt.decode(
                    access_token, app.config['OIDC_PUBLIC_KEY'],
                    algorithms='RS256',
                    audience=app.config['OIDC_CLIENT_ID']
                )
                id = (decodentoken['preferred_username'])
                headers['Sudo'] = id

            else:
                logger.debug("No authorization header, returning empty auth headers")
                headers.pop('Sudo', None)

            return f(self, request, headers)
        return decorated_function
    return decorator


class GitlabGeneric(BaseProcessor):

    @authorize()
    def process(self, request, headers):
        self.endpoint = self.endpoint.format(**app.config) + self.path
        return super().process(request, headers)


class GitlabProjects(BaseProcessor):

    @authorize()
    def process(self, request, headers):
        endpoint = self.endpoint.format(**app.config)
        if 'Sudo' in headers:
            project_response = requests.request(
                request.method,
                endpoint,
                headers=headers,
                data=request.data,
                stream=True,
                timeout=300
            )
            projects_list = project_response.json()
            return_project = json.dumps([parse_project(headers, x) for x in projects_list])
            return Response(return_project, project_response.status_code)

        else:
            response = json.dumps("No authorization header found")
            return Response(response, status=401)
