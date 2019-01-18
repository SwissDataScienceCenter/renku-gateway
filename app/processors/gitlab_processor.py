from app.processors.base_processor import BaseProcessor
from .. import app
from app.helpers.gitlab_parsers import parse_project
from urllib.parse import quote, urljoin

import logging
import json
import requests
import re

from werkzeug.datastructures import Headers
from quart import Response


logger = logging.getLogger(__name__)


SPECIAL_ROUTE_RULES = [
    {
        'before': 'repository/files/',
        'after': '/raw'
    },
    {
        'before': 'repository/files/',
        'after': ''
    },
    {
        'before': 'groups/',
        'after': '/projects'
    },
    {
        'before': 'groups/',
        'after': '/subgroups'
    },
    {
        'before': 'groups/',
        'after': ''
    },
]

SPECIAL_ROUTE_REGEXES = [
    '(.*)({before})(.*)({after})(.*)'.format(before=rule['before'], after=rule['after']) for rule in SPECIAL_ROUTE_RULES
]

GITLAB_FORWARDED_RESPONSE_HEADERS = [
    'Link',
    'X-Next-Page',
    'X-Page',
    'X-Per-Page',
    'X-Prev-Page',
    'X-Total',
    'X-Total-Pages'
]


def urlencode_paths(path):
    """Urlencode some paths before forwarding requests."""
    for rule_regex in SPECIAL_ROUTE_REGEXES:
        m = re.search(rule_regex, path)
        if m:
            return '{leading}{before}{match}{after}{trailing}'.format(
                leading=m.group(1),
                before=m.group(2),
                match=quote(m.group(3), safe=[]),
                after=m.group(4),
                trailing=m.group(5)
            )
    return path


def fix_link_header(headers):
    """Replace the GitLab URL in the Link header."""

    if 'Link' in headers:
        headers['Link'] = headers['Link'].replace(
            app.config['GITLAB_URL'], app.config['HOST_NAME']
        )

    return headers


class GitlabGeneric(BaseProcessor):

    def __init__(self, path, endpoint):
        super().__init__(path, endpoint)
        self.forwarded_headers += GITLAB_FORWARDED_RESPONSE_HEADERS


    async def process(self, request, headers):
        # Gitlab has routes where the resource identifier can include slashes
        # which must be url-encoded. We list these routes individually and re-encode
        # slashes which have been unencoded by uWSGI.
        self.path = urlencode_paths(self.path)

        self.endpoint = urljoin(self.endpoint.format(**app.config), self.path)
        access_token = headers.pop('Renku-Token', '')
        resp = await super().process(request, headers)

        if resp.status_code == 401 and access_token:  # Token has expired or is revoked
            new_token = get_gitlab_refresh_token(access_token)
            headers['Authorization'] = "Bearer {}".format(new_token)
            return await super().process(request, headers)  # retry

        return resp


    def create_response_headers(self, response):
        headers = super().create_response_headers(response)
        headers = fix_link_header(headers)
        return headers


# Note: This specific processor which is supposed to parse giblab projects is
#       not yet used and therefore commented out.

# class GitlabProjects(BaseProcessor):
#
#     async def process(self, request, headers):
#         endpoint = self.endpoint.format(**app.config)
#         if 'Authorization' in headers:
#             request_data = await request.data
#
#             project_response = requests.request(
#                 request.method,
#                 endpoint,
#                 headers=headers,
#                 data=request_data,
#                 stream=True,
#                 timeout=300
#             )
#             projects_list = project_response.json()
#             logger.debug("Gitlab response: {}".format(projects_list))
#             if request.method == 'POST':
#                 return_project = json.dumps(parse_project(headers, projects_list))
#             else:
#                 return_project = json.dumps([parse_project(headers, x) for x in projects_list])
#             return Response(return_project, project_response.status_code)
#
#         else:
#             response = json.dumps("No authorization header found")
#             return Response(response, status=401)
