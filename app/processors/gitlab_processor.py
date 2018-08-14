from app.processors.base_processor import BaseProcessor
from .. import app
from app.helpers.gitlab_parsers import parse_project
from urllib.parse import quote_plus, urljoin

import logging
import json
import requests
import re

from flask import Response


logger = logging.getLogger(__name__)


SPECIAL_ROUTE_RULES = [
    {
        'before': 'repository/files/',
        'after': '/raw'
    },
    {
        'before': 'repository/files/',
        'after': ''
    }
]

SPECIAL_ROUTE_REGEXES = [
    '(.*)({before})(.*)({after})(.*)'.format(before=rule['before'], after=rule['after']) for rule in SPECIAL_ROUTE_RULES
]


def urlencode_paths(path):
    """Urlencode some paths before forwarding requests."""
    for rule_regex in SPECIAL_ROUTE_REGEXES:
        m = re.search(rule_regex, path)
        if m:
            return '{leading}{before}{match}{after}{trailing}'.format(
                leading=m.group(1),
                before=m.group(2),
                match=quote_plus(m.group(3)),
                after=m.group(4),
                trailing=m.group(5)
            )
    return path


class GitlabGeneric(BaseProcessor):

    def process(self, request, headers):
        # Gitlab has routes where the resource identifier can include slashes
        # which must be url-encoded. We list these routes individually and re-encode
        # slashes which have been unencoded by uWSGI.
        self.path = urlencode_paths(self.path)

        self.endpoint = urljoin(self.endpoint.format(**app.config), self.path)
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
            logger.debug("Gitlab response: {}".format(projects_list))
            if request.method == 'POST':
                return_project = json.dumps(parse_project(headers, projects_list))
            else:
                return_project = json.dumps([parse_project(headers, x) for x in projects_list])
            return Response(return_project, project_response.status_code)

        else:
            response = json.dumps("No authorization header found")
            return Response(response, status=401)
