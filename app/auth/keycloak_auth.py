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
import re

from oic.oauth2.grant import Token
from .web import client

logger = logging.getLogger(__name__)


class KeycloakAccessToken():

    def process(self, request, headers):

        m = re.search(r'bearer (?P<token>.+)', headers.get('Authorization', ''), re.IGNORECASE)

        if m and jwt.decode(m.group('token'), verify=False).get('typ') in ['Offline', 'Refresh']:
            logger.debug("Swapping the refresh token for an access token")
            to = Token(resp={'refresh_token': m.group('token')})
            res = client.do_access_token_refresh(token=to)
            headers['Authorization'] = "Bearer {}".format(res.get('access_token'))

        return headers
