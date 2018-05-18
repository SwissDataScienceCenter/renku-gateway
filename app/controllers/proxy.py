# -*- coding: utf-8 -*-
#
# Copyright 2017 - Swiss Data Science Center (SDSC)
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
"""Proxy controller."""

import logging
import json
import requests
from flask import request, Response

from .. import app
from ..settings import settings
from flask_cors import CORS
import jwt


logger = logging.getLogger(__name__)
CORS(app)
g = settings()

#CHUNK_SIZE = 1024 # is this needed


# TODO use token
# def with_tokens(f):
#      """Function decorator to ensure we have OIDC tokens"""
#      return 0


@app.route('/api/projects', methods=['GET'])
def map_project() :
    logger.debug('projects controller')

    headers = dict(request.headers)

    del headers['Host']

    auth_headers = authorize(headers, g)
    if True:
   # if auth_headers!=[] :

        project_url = g['GITLAB_URL'] + "/api/v4/projects"
        project_response = requests.request(request.method, project_url, headers=headers, data=request.data, stream=True, timeout=300)
        projects_list = project_response.json()

        return_project = json.dumps([parse_project(headers, x) for x in projects_list])

        return Response(return_project, project_response.status_code)

    else:
        response = json.dumps("No authorization header found")
        return Response(response, status=401)



@app.route('/api/<path:path>', methods=['GET', 'POST', 'PUT', 'DELETE'])
def pass_through(path):
    logger.debug(path)


    headers = dict(request.headers)

    del headers['Host']

    auth_headers = authorize(headers, g)

    #if auth_headers!=[] :
    if True:
     # Respond to requester
         url = g['GITLAB_URL'] + "/api/v4/" + path
         response = requests.request(request.method, url, headers=headers, data=request.data, stream=True, timeout=300)
         logger.debug('Response: {}'.format(response.status_code))

         return Response(generate(response), response.status_code)

    else:
        response = json.dumps("No authorization header found")
        return Response(response, status=401)


def authorize(headers, g):
    if 'Authorization' in headers:

        access_token = headers.get('Authorization')[7:]
        del headers['Authorization']
        headers['Private-Token'] = g['GITLAB_PASS']

        # Get keycloak public key
        key_cloak_url = '{base}'.format(base=g['KEYCLOAK_URL'])
        token_request = json.loads(requests.get(key_cloak_url+"/auth/realms/Renga").text)

        keycloak_public_key = '-----BEGIN PUBLIC KEY-----\n' + token_request.get('public_key') + '\n-----END PUBLIC KEY-----'

        # Decode token to get user id
        decodentoken = jwt.decode(access_token, keycloak_public_key, algorithms='RS256', audience='renga-ui')
        id = (decodentoken['preferred_username'])
        headers['Sudo'] = id
        logger.debug(headers)

        return headers

    else:
        return []

def generate(response):
    for c in response.iter_lines():
        logger.debug(c)
        yield c + "\r".encode()
    return(response)


def get_readme(headers, projectid):

    readme_url = g['GITLAB_URL'] + "/api/v4/projects/" + str(projectid) + "/repository/files/README.md/raw?ref=master"
    logger.debug("Getting readme for project with  {0}". format(projectid) )

    return requests.request(request.method, readme_url, headers=headers, data=request.data, stream=True, timeout=300)

def get_kus(headers, projectid):
    issue_url = g['GITLAB_URL'] + "/api/v4/projects/" + str(projectid) + "/issues?scope=all"
    logger.debug("Getting issues for project with id {0}". format(projectid) )

    return requests.request(request.method, issue_url, headers=headers, data=request.data, stream=True, timeout=300)

def parse_kus(json_kus):


    return [parse_ku(ku) for ku in json_kus]

def parse_ku(ku):
    kuid = ku['id']

    return {
        'project_id': ku['project_id'],
        'display': {
            'title': ku['title'],
            'slug': ku['iid'],
            'display_id': ku['iid'],
            'short_description': ku['title']
        },
        'metadata':{
            'author': ku['author'],
            'created_at': ku['created_at'],
            'updated_at': ku['updated_at']
        },
        'description':['description'],
        'labels': ku['labels'],
        'notes': [],
        'assignees': ku['assignees'],
        'reactions': []
    }


def parse_project(headers, project):
    projectid = project['id']
    readme = get_readme(headers, projectid)

    kus = get_kus(headers, projectid)
    return {
        'display': {
            'title': project['name'],
            'slug': project['path'],
            'display_id': project['path_with_namespace'],
            'short_description': project['description']
        },
        'metadata': {
            'author': project['owner'],
            'created_at': project['created_at'],
            'last_activity_at': project['last_activity_at'],
            'permissions': [],
            'id': projectid
        },
        'description': project['description'],
        'long_description': readme.text,
        'name': project['name'],
        'forks_count': project['forks_count'],
        'star_count': project['star_count'],
        'tags': project['tag_list'],
        'kus': parse_kus(kus.json()),
        'repository_content': []
    }

def get_notes(headers, projectid, issueid):

    notes_url = g['GITLAB_URL'] + "/api/v4/projects/" + str(projectid) + "/issues/" + str(issueid) + "/notes"
    logger.debug("Getting notes for issue with id {0} in project with id {1}".format(issueid, projectid))

    return requests.request(request.method, notes_url, headers=headers, data=request.data, stream=True, timeout=300)
