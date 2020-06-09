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
import requests
import json

PROVIDER_KINDS = {
    "JUPYTERHUB": "jupyterhub",
    "GITLAB": "gitlab",
    "KEYCLOAK": "keycloak",
}


class OAuthProviderApp:
    """A simple class combining some information about the oauth provider and the
    application registered with the provider."""

    def __init__(
        self,
        kind=None,
        base_url=None,
        client_id=None,
        client_secret=None,
        authorization_endpoint=None,
        token_endpoint=None,
    ):
        self.kind = kind
        self.base_url = base_url
        self.client_id = client_id
        self.client_secret = client_secret
        self.authorization_endpoint = authorization_endpoint
        self.token_endpoint = token_endpoint

    # TODO: Use marshmallow for (de)serialization
    # TODO: https://github.com/SwissDataScienceCenter/renku-gateway/issues/231
    def to_json(self):

        SERIALIZER_ATTRIBUTES = [
            "kind",
            "base_url",
            "client_id",
            "client_secret",
            "authorization_endpoint",
            "token_endpoint",
        ]
        provder_app_dict = {key: vars(self)[key] for key in SERIALIZER_ATTRIBUTES}
        return json.dumps(provder_app_dict)

    @classmethod
    def from_dict(cls, provider_app_dict):
        return _typecast_provider_app(cls(**provider_app_dict))

    @classmethod
    def from_json(cls, serialized_provider_app):
        return cls.from_dict(json.loads(serialized_provider_app))


class GitLabProviderApp(OAuthProviderApp):
    def __init__(self, base_url=None, client_id=None, client_secret=None):
        super().__init__(
            kind=PROVIDER_KINDS["GITLAB"],
            base_url=base_url,
            client_id=client_id,
            client_secret=client_secret,
            authorization_endpoint="{}/oauth/authorize".format(base_url),
            token_endpoint="{}/oauth/token".format(base_url),
        )


class JupyterHubProviderApp(OAuthProviderApp):
    def __init__(self, base_url=None, client_id=None, client_secret=None):
        super().__init__(
            kind=PROVIDER_KINDS["JUPYTERHUB"],
            base_url=base_url,
            client_id=client_id,
            client_secret=client_secret,
            authorization_endpoint="{}/hub/api/oauth2/authorize".format(base_url),
            token_endpoint="{}/hub/api/oauth2/token".format(base_url),
        )


class KeycloakProviderApp(OAuthProviderApp):
    def __init__(self, base_url=None, client_id=None, client_secret=None):
        super().__init__(
            kind=PROVIDER_KINDS["KEYCLOAK"],
            base_url=base_url,
            client_id=client_id,
            client_secret=client_secret,
        )
        # Fetch the details from Keycloak itself
        self.get_config()

    def get_config(self):
        """Get the endpoints from the base URL by querying keycloak directly."""
        keycloak_config = requests.get(
            "{}/.well-known/openid-configuration".format(self.base_url)
        ).json()
        self.authorization_endpoint = keycloak_config["authorization_endpoint"]
        self.token_endpoint = keycloak_config["token_endpoint"]

    # TODO: Keycloak public key / realm information could be added here.


def _typecast_provider_app(provider_app):
    """Cast an OAuthProviderApp object to the correct subclass."""
    if provider_app.kind == "gitlab":
        provider_app.__class__ = GitLabProviderApp
    if provider_app.kind == "keycloak":
        provider_app.__class__ = KeycloakProviderApp
    if provider_app.kind == "jupyterhub":
        provider_app.__class__ = JupyterHubProviderApp
    return provider_app
