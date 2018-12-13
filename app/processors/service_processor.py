from urllib.parse import urljoin

from app.processors.base_processor import BaseProcessor

from .. import app


class ServiceGeneric(BaseProcessor):
    async def process(self, request, header):
        self.endpoint = urljoin(self.endpoint.format(**app.config), self.path)
        return await super().process(request, header)
