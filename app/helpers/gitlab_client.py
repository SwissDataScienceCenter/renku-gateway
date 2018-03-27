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
from flask import request
import logging
from .. import app

logger = logging.getLogger(__name__)

def get_readme(headers, projectid):
    readme_url = app.config['GITLAB_URL'] + "/api/v4/projects/" + str(projectid) + "/repository/files/README.md/raw?ref=master"
    logger.debug("Getting readme for project with {0}". format(projectid) )

    return requests.request(request.method, readme_url, headers=headers, data=request.data, stream=True, timeout=300)


def get_kus(headers, projectid):
    ku_url = app.config['GITLAB_URL'] + "/api/v4/projects/" + str(projectid) + "/issues?scope=all"
    logger.debug("Getting issues for project with id {0}". format(projectid) )

    return requests.request(request.method, ku_url, headers=headers, data=request.data, stream=True, timeout=300)
