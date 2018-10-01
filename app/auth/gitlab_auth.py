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
import jwt
import logging


from .. import app
from app.helpers.gitlab_user_utils import get_or_create_gitlab_user

logger = logging.getLogger(__name__)


class GitlabSudoToken():

    def process(self, request, headers):

        if 'Authorization' in headers:
            # logger.debug('Authorization header present, sudo token exchange')
            # logger.debug('outgoing headers: {}'.format(json.dumps(headers))

            # TODO: Use regular expressions to extract the token from the header
            access_token = headers.get('Authorization')[7:]
            del headers['Authorization']
            headers['Private-Token'] = app.config['GITLAB_PASS']

            # Decode token to get user id
            # TODO: What happens if the validation of the token fails for other reasons?
            decodentoken = jwt.decode(
                access_token, app.config['OIDC_PUBLIC_KEY'],
                algorithms='RS256',
                audience=app.config['OIDC_CLIENT_ID']
            )
            headers['Sudo'] = get_or_create_gitlab_user(decodentoken)
        else:
            # logger.debug("No authorization header, returning empty auth headers")
            headers.pop('Sudo', None)

        return headers
