from app.processors.base_processor import BaseProcessor
from .. import app
from app.helpers.gitlab_parsers import parse_project

import logging
import json
import requests

from flask import Response


logger = logging.getLogger(__name__)


class GitlabGeneric(BaseProcessor):

    def process(self, request, headers):
        self.endpoint = self.endpoint.format(**app.config) + self.path
        return super().process(request, headers)


class GitlabProjects(BaseProcessor):

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
