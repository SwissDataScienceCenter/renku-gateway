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

from quart import Blueprint, Response, current_app
from SPARQLWrapper import DIGEST, JSONLD, POST, SPARQLWrapper

blueprint = Blueprint('graph', __name__, url_prefix='/graph')

LINEAGE_GLOBAL = """
PREFIX prov: <http://www.w3.org/ns/prov#>
PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX wfdesc: <http://purl.org/wf4ever/wfdesc#>
PREFIX wf: <http://www.w3.org/2005/01/wf/flow#>
PREFIX wfprov: <http://purl.org/wf4ever/wfprov#>
PREFIX foaf: <http://xmlns.com/foaf/0.1/>
PREFIX dcterms: <http://purl.org/dc/terms/>

CONSTRUCT {{
  ?v rdfs:label ?label_v .
  ?w prov:used ?v ;
     rdfs:label ?label_w ;
     prov:hadRole ?role .
}} WHERE {{
  {{
    SELECT ?entity
    WHERE {{
      {filter}
    }}
    GROUP BY ?entity
  }}
  {{
    SELECT (?entity AS ?v) (?activity AS ?w) ?role (?comment AS ?label_w)
           (?path AS ?label_v)
    WHERE {{
      ?activity (prov:qualifiedUsage/prov:entity) ?entity .
      ?entity prov:atLocation ?path .
      ?activity prov:qualifiedUsage ?qual .
      ?qual prov:hadRole ?role .
      ?qual prov:entity ?entity .
      ?qual rdf:type ?type .
      ?activity rdf:type wfprov:ProcessRun .
      ?activity rdfs:comment ?comment .
      FILTER NOT EXISTS {{ ?activity rdf:type wfprov:WorkflowRun }}
    }}
  }} UNION {{
    SELECT (?activity AS ?v) (?entity AS ?w) ?role (?comment AS ?label_v)
           (?path AS ?label_w)
    WHERE {{
      ?entity (prov:qualifiedGeneration/prov:activity) ?activity .
      ?entity prov:qualifiedGeneration ?qual ;
              prov:atLocation ?path .
      ?qual prov:hadRole ?role .
      ?qual prov:activity ?activity .
      ?qual rdf:type ?type .
      ?activity rdf:type wfprov:ProcessRun ;
      rdfs:comment ?comment .
      FILTER NOT EXISTS {{ ?activity rdf:type wfprov:WorkflowRun }}
    }}
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
    sparql.setReturnFormat(JSONLD)
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

    return Response(
        sparql.query(),
        content_type='application/ld+json',
    )
