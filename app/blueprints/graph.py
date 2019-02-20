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
"""Graph endpoint."""

from quart import Blueprint, Response, current_app, jsonify
from SPARQLWrapper import DIGEST, JSON, POST, XML, SPARQLWrapper

blueprint = Blueprint('graph', __name__, url_prefix='/graph')

LINEAGE_GLOBAL = """
PREFIX prov: <http://www.w3.org/ns/prov#>
PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
PREFIX wfdesc: <http://purl.org/wf4ever/wfdesc#>
PREFIX wf: <http://www.w3.org/2005/01/wf/flow#>
PREFIX wfprov: <http://purl.org/wf4ever/wfprov#>
PREFIX foaf: <http://xmlns.com/foaf/0.1/>
PREFIX dcterms: <http://purl.org/dc/terms/>

SELECT ?v ?w
WHERE {{
  {{
    SELECT ?entity
    WHERE {{
      {filter}
    }}
    GROUP BY ?entity
  }}
  {{
    ?entity prov:qualifiedGeneration/prov:activity ?activity .
    BIND (?entity AS ?v)
    BIND (?activity AS ?w)
  }} UNION {{
    ?activity prov:qualifiedUsage/prov:entity ?entity .
    BIND (?activity AS ?v)
    BIND (?entity AS ?w)
  }}
}}
"""


@blueprint.route('/<namespace>/<project>/lineage')
@blueprint.route('/<namespace>/<project>/lineage/<commit_ish>')
@blueprint.route('/<namespace>/<project>/lineage/<commit_ish>/<path:path>')
async def lineage(namespace, project, commit_ish=None, path=None):
    """Query graph service."""
    gitlab_url = current_app.config['GITLAB_URL']
    if gitlab_url.endswith('/gitlab'):
        gitlab_url = gitlab_url[:-len('/gitlab')]
    central_node = None
    project_url = '{gitlab}/{namespace}/{project}'.format(
        gitlab=gitlab_url,
        namespace=namespace,
        project=project,
    )

    sparql = SPARQLWrapper(current_app.config['SPARQL_ENDPOINT'])

    # SPARQLWrapper2 for JSON

    sparql.setHTTPAuth(DIGEST)
    sparql.setCredentials(
        current_app.config['SPARQL_USERNAME'],
        current_app.config['SPARQL_PASSWORD'],
    )
    sparql.setReturnFormat(JSON)
    sparql.setMethod(POST)

    filter = [
        '?entity dcterms:isPartOf ?project .',
        'FILTER (?project = <{project_url}>)'.format(project_url=project_url),
    ]
    if commit_ish:
        filter.extend([
            '?entity (prov:qualifiedGeneration/prov:activity | '
            '^prov:entity/^prov:qualifiedUsage) ?qactivity .',
            'FILTER (?qactivity = <file:///commit/{commit_ish}>)'.format(
                commit_ish=commit_ish
            ),
        ])

    if path:
        central_node = 'file:///blob/{commit_ish}/{path}'.format(
            commit_ish=commit_ish,
            path=path,
        )
        filter.append(
            'FILTER (?entity = <{central_node}>)'.format(
                central_node=central_node
            ),
        )

    query = LINEAGE_GLOBAL.format(filter='\n          '.join(filter))

    sparql.setQuery(query)
    results = sparql.query().convert()

    node_ids = set()
    edges = []

    for item in results['results']['bindings']:
        node_ids |= set(value['value'] for value in item.values())
        edges.append({key: value['value'] for key, value in item.items()})

    return jsonify({
        'nodeIds': list(node_ids),
        'edges': edges,
        'centralNode': central_node,
    })
