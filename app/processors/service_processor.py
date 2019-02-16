from urllib.parse import urljoin

from quart import current_app
from app.processors.base_processor import BaseProcessor


class ServiceGeneric(BaseProcessor):
    async def process(self, request, header):
        self.endpoint = urljoin(self.endpoint.format(**current_app.config), self.path)
        return await super().process(request, header)
