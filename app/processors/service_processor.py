from .. import app
from app.processors.base_processor import BaseProcessor
from urllib.parse import urljoin


class ServiceGeneric(BaseProcessor):

    def process(self, request, header):
        self.endpoint = urljoin(self.endpoint.format(**app.config), self.path)
        return super().process(request, header)
