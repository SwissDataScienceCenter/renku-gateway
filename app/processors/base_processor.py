import json
import logging
import requests

from flask import Response
from werkzeug.datastructures import Headers

logger = logging.getLogger(__name__)


class BaseProcessor:

    def __init__(self, path, endpoint):
        self.path = path
        self.endpoint = endpoint
        self.forwarded_headers = [
            'Content-Type',
        ]
        logger.debug('Processor with path = "{}" and endpoint = "{}"'.format(path, endpoint))

    def process(self, request, headers):
        logger.debug('Request path: {}'.format(self.path))
        logger.debug('Forward endpoint: {}'.format(self.endpoint))
        logger.debug('incoming headers: {}'.format(json.dumps(headers)))

        # Respond to requester
        response = requests.request(
            request.method,
            self.endpoint,
            headers=headers,
            params=request.args,
            data=request.data,
            stream=True,
            timeout=300
        )

        logger.debug('Response: {}'.format(response.status_code))
        logger.debug('Response headers: {}'.format(response.headers))

        return Response(
            response=self.generate_response_data(response),
            headers=self.create_response_headers(response),
            status=response.status_code,
        )

    def generate_response_data(self, response):
        for c in response.iter_lines():
            # logger.debug(c)
            yield c + "\r\n".encode()
        yield "\r\n".encode()

    def create_response_headers(self, response):
        headers = Headers()
        for header in response.headers:
            if header in self.forwarded_headers:
                headers.add(header, response.headers[header])
        return headers
