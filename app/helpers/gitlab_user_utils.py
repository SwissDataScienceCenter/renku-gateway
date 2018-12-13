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
"""Helpers for dealing with the GitLab user data."""

import json
import logging
import re

import requests
from quart import current_app

logger = logging.getLogger(__name__)

# A dictionary to cache GitLab usernames given the "sub" claim from the keycloak access token
# as a key. This dictionary can be trashed without any functional implications, it will just
# result in a few extra queries to GitLab.
GITLAB_USERNAMES = {}


def get_or_create_gitlab_user(access_token):
    """Get the username of a a user given the validated JWT keycloak access_token. Create
    a new user in case it doesn't already exist in GitLab."""

    username = GITLAB_USERNAMES.get(access_token['sub'], None)
    if username:
        return username

    sudo_header = {'Private-Token': current_app.config['GITLAB_PASS']}

    query_params = {
        'extern_uid': access_token['sub'],
        'provider': 'oauth2_generic'
    }
    user_response = requests.get(
        current_app.config['GITLAB_URL'] + '/api/v4/users',
        headers=sudo_header,
        params=query_params
    )

    # More than one user found -> should not happen
    if len(user_response.json()) > 1:
        logging.error(
            'More than one user with ' +
            'extern_uid={} for provider oauth2_generic.'.
            format(access_token['sub'])
        )
        return None

    # No user found, lets create it.
    # We emulate the behaviour of gitlab in creating the username from the email
    # address, while appending integers in case a username is already taken.
    elif len(user_response.json()) == 0:

        username_counter = 0
        while True:
            username_appendix = '' if username_counter == 0 else str(
                username_counter
            )
            username_base = re.match(
                r'[a-zA-Z0-9\.\_\-]*', access_token['preferred_username']
            ).group(0)

            body = {
                'username': username_base + username_appendix,
                'email': access_token['email'],
                'name':
                    '{first} {last}'.format(
                        first=access_token['given_name'],
                        last=access_token['family_name']
                    ),
                'extern_uid': access_token['sub'],
                'provider': 'oauth2_generic',
                'skip_confirmation': True,
                'reset_password': True
            }

            new_user_response = requests.post(
                current_app.config['GITLAB_URL'] + '/api/v4/users',
                headers=sudo_header,
                data=body
            )
            if (
                new_user_response.status_code != 409 or
                new_user_response.json()['message'] !=
                'Username has already been taken'
            ):
                break
            username_counter += 1

        if new_user_response.status_code != 201:
            logging.error(
                'Problem creating user from body {}'.format(json.dumps(body))
            )
            logging.error(new_user_response.json()['message'])

        username = new_user_response.json()['username']

    # Exactly one user found, return the username
    else:
        username = user_response.json()[0]['username']

    GITLAB_USERNAMES[access_token['sub']] = username
    return username
