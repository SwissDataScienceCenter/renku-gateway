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
"""Graph endpoint."""

from SPARQLWrapper import DIGEST, JSON, POST, SPARQLWrapper, XML
from quart import Blueprint, Response, current_app, jsonify

blueprint = Blueprint('graph', __name__)

LINEAGE_GLOBAL = """
PREFIX prov: <http://www.w3.org/ns/prov#>
PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
PREFIX wfdesc: <http://purl.org/wf4ever/wfdesc#>
PREFIX wf: <http://www.w3.org/2005/01/wf/flow#>
PREFIX wfprov: <http://purl.org/wf4ever/wfprov#>
PREFIX foaf: <http://xmlns.com/foaf/0.1/>

SELECT DISTINCT ?node
WHERE {{
  ?node rdf:type prov:Entity ;
        prov:atLocation ?project ;
        (prov:qualifiedGeneration/prov:activity) ?activity .

  {filter}

}}
"""

response = {
    "nodeIds": [
        "5e733:/data/zh/standardized.csv", "5e733:",
        "cab58:/data/zh/homog_mo_SMA.txt"
    ],
    "edges": [{
        "v": "cab58:/data/zh/homog_mo_SMA.txt",
        "w": "5e733:"
    }, {
        "v": "5e733:",
        "w": "5e733:/data/zh/standardized.csv"
    }],
    "centralNode":
    "cab58:/data/zh/homog_mo_SMA.txt"
}


@blueprint.route('/<namespace>/<project>/lineage')
@blueprint.route('/<namespace>/<project>/lineage/<commit_ish>')
@blueprint.route('/<namespace>/<project>/lineage/<commit_ish>/<path:path>')
async def lineage(namespace, project, commit_ish=None, path=None):
    """Query graph service."""
    central_node = None
    project_url = '{gitlab}/{namespace}/{project}'.format(
        gitlab=current_app.config['GITLAB_URL'],
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
        'FILTER (?project = <{project_url}>)'.format(project_url=project_url),
    ]
    if commit_ish:
        filter.append(
            'FILTER (?activity = <file:///{commit_ish}#>)'.format(
                commit_ish=commit_ish
            ),
        )

    if path:
        central_node = 'file:///{commit_ish}/{path}'.format(
            commit_ish=commit_ish,
            path=path,
        )
        filter.append(
            'FILTER (?node = <{central_node}>)'.format(
                central_node=central_node
            ),
        )

    query = LINEAGE_GLOBAL.format(filter='\n  '.join(filter))

    sparql.setQuery(query)
    results = sparql.query().convert()

    node_ids = [
        item['node']['value'] for item in results['results']['bindings']
    ]

    return jsonify({
        'nodeIds': node_ids,
        'centralNode': central_node,
    })
