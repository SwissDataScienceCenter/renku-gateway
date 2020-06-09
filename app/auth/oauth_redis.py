# -*- coding: utf-8 -*-
#
# Copyright 2018 - Swiss Data Science Center (SDSC)
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

"""
This module handles the storing of user- and provider specific oauth
client instances in redis.
"""

import base64
from cryptography.fernet import Fernet
import sys

from flask import current_app
from oauthlib.oauth2.rfc6749.errors import OAuth2Error

from .oauth_client import RenkuWebApplicationClient

if "pytest" in sys.modules:
    from fakeredis import FakeStrictRedis as StrictRedis
else:
    from redis import StrictRedis


def create_fernet_key(hex_key):
    """Small helper to transform a standard 64 hex character secret
    into the required urlsafe base64 encoded 32-bytes which serve
    as fernet key."""

    # Check if we have 32 bytes in hex form
    if not len(hex_key) == 64:
        raise ValueError("provided key must be 64 characters: {}".format(hex_key))
    try:
        int(hex_key, 16)
    except ValueError:
        raise ValueError("provided key contains non-hex character: {}".format(hex_key))

    # Convert
    return base64.urlsafe_b64encode(
        bytes([int(hex_key[i : i + 2], 16) for i in range(0, len(hex_key), 2)])
    )


class OAuthRedis(StrictRedis):
    """Just a regular StrictRedis store with extra methods for
    setting and getting encrypted serializations of oauth client objects."""

    def __init__(self, *args, hex_key=None, **kwargs):
        super().__init__(*args, **kwargs)
        self.fernet = Fernet(create_fernet_key(hex_key))

    def set_enc(self, name, value):
        """Set method with encryption."""
        return super().set(name, self.fernet.encrypt(value))

    def get_enc(self, name):
        """Get method with decryption."""
        return self.fernet.decrypt(super().get(name))

    def set_oauth_client(self, name, oauth_client, **kwargs):
        """Put a client object into the store."""
        return self.set_enc(name, oauth_client.to_json().encode())

    def get_oauth_client(self, name, no_refresh=False, **kwargs):
        """Get a client object from the store, refresh if necessary."""
        oauth_client = RenkuWebApplicationClient.from_json(self.get_enc(name).decode())

        # We refresh 5 seconds before the token/client actually expires
        # to avoid unlucky edge cases.
        if not no_refresh and oauth_client.expires_soon():
            try:
                # TODO: Change logger to have no dependency on the current_app here.
                # https://github.com/SwissDataScienceCenter/renku-gateway/issues/113
                current_app.logger.info("Refreshing {}".format(name))
                oauth_client.refresh_access_token()
                self.set_enc(name, oauth_client.to_json().encode(), **kwargs)
            except OAuth2Error as e:
                current_app.logger.warn(
                    "Error refreshing tokens: {} "
                    "Clearing client from redis.".format(e.error)
                )
                self.delete(name)
                return None

        return oauth_client
