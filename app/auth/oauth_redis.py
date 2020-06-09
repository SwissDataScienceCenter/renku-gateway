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

from cryptography.fernet import Fernet
import sys
import time

from flask import current_app
from oauthlib.oauth2.rfc6749.errors import OAuth2Error

from .oauth_client import RenkuWebApplicationClient

if "pytest" in sys.modules:
    from fakeredis import FakeStrictRedis as StrictRedis
else:
    from redis import StrictRedis


class OAuthRedis(StrictRedis):
    """Just a regular StrictRedis store with extra methods for
    setting and getting encrypted serializations of oauth client objects."""

    def __init__(self, *args, fernet_key=None, **kwargs):
        super().__init__(*args, **kwargs)
        self.fernet = Fernet(fernet_key)

    def set_enc(self, name, value, **kwargs):
        """Set method with encryption."""
        return super().set(name, self.fernet.encrypt(value), **kwargs)

    def get_enc(self, name, **kwargs):
        """Get method with decryption."""
        return self.fernet.decrypt(super().get(name, **kwargs))

    def set_oauth_client(self, name, oauth_client, **kwargs):
        """Put a client object into the store."""
        return self.set_enc(name, oauth_client.to_json().encode(), **kwargs)

    def get_oauth_client(self, name, no_refresh=False, **kwargs):
        """Get a client object from the store, refresh if necessary."""
        oauth_client = RenkuWebApplicationClient.from_json(
            self.get_enc(name, **kwargs).decode()
        )

        # We refresh 5 seconds before the token/client actually expires
        # to avoid unlucky edge cases.
        if (
            not no_refresh
            and oauth_client._expires_at
            and oauth_client._expires_at < time.time() - 5
        ):
            try:
                # TODO: Change logger to have no dependency on the current_app here.
                # TODO: https://github.com/SwissDataScienceCenter/renku-gateway/issues/113
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
