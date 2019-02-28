# -*- coding: utf-8 -*-
#
# Copyright 2018-2019 - Swiss Data Science Center (SDSC)
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
"""Webhooks endpoints."""

from quart import Blueprint, current_app, request

from app.processors.service_processor import ServiceGeneric
from app.gateway.proxy import pass_through
from app.auth import GitlabUserToken

from . import all_methods


blueprint = Blueprint('webhooks', __name__)


@blueprint.route('/projects/<project_id>/graph-hooks/<path:path>',
                 methods=all_methods)
@blueprint.route('/projects/<project_id>/graph-hooks', methods=ALL_HTTP_METHODS)
async def forward_to_webooks(project_id, path=''):
    path = '/projects/{}/webhooks/{}'.format(project_id, path).rstrip('/')
    processor = ServiceGeneric(
        path,
        '{}/'.format(current_app.config['WEBHOOK_SERVICE_HOSTNAME'])
    )
    return await pass_through(
        request,
        processor,
        GitlabUserToken(header_field='OAUTH-TOKEN', header_prefix='')
    )
