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

        rsp = Response(self.generate(response), response.status_code)
        # rsp.headers = Headers(response.headers.lower_items())  # this seems to break things sometimes

        logger.debug('Response: {}'.format(response.status_code))
        logger.debug('Response headers: {}'.format(rsp.headers))

        return rsp

    def generate(self, response):
        for c in response.iter_lines():
            # logger.debug(c)
            yield c + "\r\n".encode()
        yield "\r\n".encode()
