from .. import app
from app.processors.base_processor import BaseProcessor
from urllib.parse import urljoin

import json
import quart
import logging
import re

logger = logging.getLogger(__name__)


class GraphGeneric(BaseProcessor):

    async def process(self, request, header):
        self.endpoint = urljoin(self.endpoint.format(**app.config), self.path)

        # Note: This regex will always match!
        m = re.search('(^[^/]*)/?([^/]*)/?(.*)', self.path)
        project_id = m.group(1)
        commit_ish = m.group(2)
        file_path = m.group(3)

        # graph = await call_graph_service(project_id, commit_ish, file_path)
        graph = json.loads('{"nodeIds":["5e733:/data/zh/standardized.csv","5e733:","cab58:/data/zh/homog_mo_SMA.txt"],"edges":[{"v":"cab58:/data/zh/homog_mo_SMA.txt","w":"5e733:"},{"v":"5e733:","w":"5e733:/data/zh/standardized.csv"}],"centralNode":"cab58:/data/zh/homog_mo_SMA.txt"}')

        return quart.jsonify(graph)
