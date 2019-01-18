import os
import json
import logging
import aiohttp

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



        async with aiohttp.ClientSession() as session:
            async with session.request(
                        request.method,
                        self.endpoint,
                        headers=headers,
                        params=request.args,
                        data=(await request.data),
                    ) as response:

                response_data = await response.read()

                logger.debug('Response: {}'.format(response.status))
                logger.debug('Response headers: {}'.format(response.headers))

                return Response(
                    # response=response_generator(response),
                    response=response_data,
                    headers=self.create_response_headers(response),
                    status=response.status
                )

    # TODO: This does not work yet.
    # TODO: Enable streaming responses at a later point.

    # async def response_generator(response):
    #     async for chunk in response.content.iter_chunked(1024):
    #         yield chunk

    def create_response_headers(self, response):
        headers = Headers()
        for header in response.headers:
            if header in self.forwarded_headers:
                headers.add(header, response.headers[header])
        return headers
