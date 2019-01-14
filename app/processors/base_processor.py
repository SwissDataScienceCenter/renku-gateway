import os
import json
import logging
import requests

from quart import Response
from werkzeug.datastructures import Headers

gateway_env = os.environ.get('GATEWAY_ENV')
dev_env = (gateway_env == 'development')
logger = logging.getLogger(__name__)


def headers_for_development(headers):
    """Return headers (dict) modified for the development env."""
    headers_to_strip = set(["X-Forwarded-Proto"])
    return {k: v for (k, v) in headers.items() if k not in headers_to_strip}


class BaseProcessor:

    def __init__(self, path, endpoint):
        self.path = path
        self.endpoint = endpoint
        self.forwarded_headers = [
            'Content-Type',
        ]
        logger.debug('Processor with path = "{}" and endpoint = "{}"'.format(path, endpoint))

    async def process(self, request, headers):
        logger.debug('Request path: {}'.format(self.path))
        logger.debug('Forward endpoint: {}'.format(self.endpoint))
        logger.debug('incoming headers: {}'.format(json.dumps(headers)))

        if dev_env:
            headers = headers_for_development(headers)
            logger.debug('development headers: {}'.format(json.dumps(headers)))

        # Respond to requester
        response = requests.request(
            request.method,
            self.endpoint,
            headers=headers,
            params=request.args,
            data=(await request.data),
            stream=True,
            timeout=30
        )

        logger.debug('Response: {}'.format(response.status_code))
        logger.debug('Response headers: {}'.format(response.headers))

        return Response(
            response=self.generate_response_data(response),
            headers=self.create_response_headers(response),
            status=response.status_code,
        )

    async def generate_response_data(self, response):
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
