# -*- coding: utf-8 -*-
#
# Copyright 2018 - Swiss Data Science Center (SDSC)
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


import requests
from quart import request
import logging
from .. import app

from app.helpers.gitlab_client import get_readme, get_kus


# Methods to parse the gitlab responses into Renku data model
# need to be implemented simultaneously with the Renku UI



logger = logging.getLogger(__name__)

def parse_kus(headers, json_kus):
    return [parse_ku(headers, ku) for ku in json_kus]


def parse_ku(headers, ku):
    kuid = ku['id']
    kuiid = ku['iid']
    projectid = ku['project_id']

    reactions_url = app.config['GITLAB_URL'] + "/api/v4/projects/" + str(projectid) + "/issues/" + str(kuid) +  "/award_emoji"
    reactions_response = (requests.request(request.method, reactions_url, headers=headers, data=request.data, stream=True, timeout=300))

    if reactions_response.status_code == 200:
        reactions = reactions_response.json()
    else:
        reactions = []

    contribution_url =  app.config['GITLAB_URL'] + "/api/v4/projects/" + str(projectid) + "/issues/" + str(kuid) + "/notes"
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

    user_url =  app.config['GITLAB_URL'] + "api/v4/users" + str(user_id)
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
