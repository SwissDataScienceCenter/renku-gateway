from .. import app
from app.processors.base_processor import BaseProcessor


class ServiceGeneric(BaseProcessor):

    def process(self, request, header):
        self.endpoint = self.endpoint.format(**app.config) + self.path
        return super().process(request, header)
