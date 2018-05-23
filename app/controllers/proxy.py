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

    if auth_headers!=[] :
     # Respond to requester
         url = g['GITLAB_URL'] + "/api/" + path
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
        key_cloak_url = '{base}'.format(base=g['OIDC_ISSUER'])
        token_request = json.loads(requests.get(key_cloak_url).text)

        keycloak_public_key = '-----BEGIN PUBLIC KEY-----\n' + token_request.get('public_key') + '\n-----END PUBLIC KEY-----'

        # Decode token to get user id
        decodentoken = jwt.decode(access_token, keycloak_public_key, algorithms='RS256', audience=g['OIDC_CLIENT_ID'])
        id = (decodentoken['preferred_username'])
        headers['Sudo'] = id
        headers['Private-Token'] = 'dummy-secret'
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
    ku_url = g['GITLAB_URL'] + "/api/v4/projects/" + str(projectid) + "/issues?scope=all"
    logger.debug("Getting issues for project with id {0}". format(projectid) )

    return requests.request(request.method, ku_url, headers=headers, data=request.data, stream=True, timeout=300)

def parse_kus(headers, json_kus):
    return [parse_ku(headers, ku) for ku in json_kus]

def parse_ku(headers, ku):
    kuid = ku['id']
    kuiid = ku['iid']
    projectid = ku['project_id']

    reactions_url = g['GITLAB_URL'] + "/api/v4/projects/" + str(projectid) + "/issues/" + str(kuid) +  "/award_emoji"
    reactions_response = (requests.request(request.method, reactions_url, headers=headers, data=request.data, stream=True, timeout=300))

    if reactions_response.status_code == 200:
        reactions = reactions_response.json()
    else:
        reactions = []

    contribution_url =  g['GITLAB_URL'] + "/api/v4/projects/" + str(projectid) + "/issues/" + str(kuid) + "/notes"
    contribution_response = (requests.request(request.method, contribution_url, headers=headers, data=request.data, stream=True, timeout=300))

    if contribution_response.status_code == 200:
        contributions = [parse_contribution(headers, x) for x in contribution_response.json()]
    else:
        contributions = []


    return {
        'project_id': projectid,
        'display': {
            'title': ku['title'],
            'slug': kuiid,
            'display_id': kuiid,
            'short_description': ku['title']
        },
        'metadata':{
            'author': ku['author'], #must be a user object
            'created_at': ku['created_at'],
            'updated_at': ku['updated_at'],
            'id': kuid,
            'iid': kuiid
        },
        'description': ku['description'],
        'labels': ku['labels'],
        'contributions': contributions,
        'assignees': ku['assignees'],
        'reactions': reactions
    }


def parse_project(headers, project):
    projectid = project['id']
    readme = get_readme(headers, projectid)

    if get_kus(headers, projectid)!= []:
        kus = parse_kus(headers, get_kus(headers, projectid).json()),
    else:
        kus = []

    return {
        'display': {
            'title': project['name'],
            'slug': project['path'],
            'display_id': project['path_with_namespace'],
            'short_description': project['description']
        },
        'metadata': {
            'author': project['owner'], # parse into user object
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
        'kus': kus,
        'repository_content': []
    }

def parse_contribution(headers, contribution):
    return {
        'ku_id': contribution['noteable_id'],
        'ku_iid': contribution['noteable_iid'],
        'metadata': {
             'author': contribution['author'], #parse_user(headers, contribution['author']['id']),
             'created_at': contribution['created_at'],
             'updated_at' : contribution['updated_at'],
             'id': contribution['id']
        },
        'body': contribution['body']
    }

def parse_user(headers, user_id):

    user_url =  g['GITLAB_URL'] + "api/v4/users" + str(user_id)
    user = (requests.request(request.method, user_url, headers=headers, data=request.data, stream=True, timeout=300)).json()

    return {
        'metadata': {
            'created_at': user['created_at'],
            'last_activity_at': user['last_activity_at'],
            'id': user['id']
         },
        'username': user['username'],
        'name': user['name'],
        'avatar_url': user['avatar_url']
    }
