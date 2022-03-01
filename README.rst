..
  Copyright 2017-2018 - Swiss Data Science Center (SDSC)
  A partnership between École Polytechnique Fédérale de Lausanne (EPFL) and
  Eidgenössische Technische Hochschule Zürich (ETHZ).

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

==================
 Renku API gateway
==================

**The Renku platform is under very active development and should be considered highly
volatile.**

The Renku API gateway connects the different Renku clients to the various Renku backend
services (GitLab, Renku components etc). It consists of two parts: a traefik reverse-proxy
(gateway) and a flask application acting predominantly as traefik forward-auth middleware 
(gateway-auth).


Developing the gateway-auth component
-------------------------------------
The renku gateway-auth component is best developped in the context of a full renku 
deployment. In order to get an instance of Renku up and running, clone the main Renku
repository and follow these instructions_.

.. _instructions: https://renku.readthedocs.io/en/latest/developer/setup.html

Once you have an instance of Renku running, you could modify the gateway code, build the 
image, re-build the chart, redeploy, etc... This will make for a poor development experience
with very long feedback cycles.

Instead we recommend intercepting traffic to the gateway-auth component and routing it to
your local machine through telepresence_ (note that currently you MUST use version 2.4.X, 
mac users see in particular tele-troubleshooting_). Once telepresence is installed, create a 
python environment and install the necessary python dependencies by running 
:code:`pipenv install --dev`. Then, create a telepresence intercept using the dedicated 
:code:`./telepresence-intercept.sh` script and follow the instructions. This will forward 
all requests to the gateway-auth service deployed in the cluster to a flask development 
server running on your local machine (with hot reloading, etc). You can now use your 
favourite IDE and develop the component completely locally. Stopping the development server
through :code:`ctrl-C` and then stopping the shell process invoked with the intercept by 
typing :code:`exit` will terminate the intercept.

.. _telepresence: https://www.telepresence.io/docs/v2.4/quick-start/
.. _tele-troubleshooting: https://www.telepresence.io/docs/latest/troubleshooting/


Tests
-----

You can run tests with

::

    $ pipenv run pytest

Configuration
-------------
The simplest way to deploy the gateway is using Helm charts_ and setting the needed values.
But if you prefer to run directly the docker image here is the list of all environment variables that can be set, with their default values.

.. _charts: helm-chart/

+---------------------------------+-----------------------------------------------------------------------------------------------------------------+----------------------------------+
| Variable name                   | Description                                                                                                     | Default value                    |
+=================================+=================================================================================================================+==================================+
| HOST_NAME                       | The URL of this service.                                                                                        | http://gateway.renku.build       |
+---------------------------------+-----------------------------------------------------------------------------------------------------------------+----------------------------------+
| GATEWAY_SECRET_KEY              | Must be exactly 64 hex characters! Used to encrypt session cookies and redis content. Must be set, no default!  | -                                |
+---------------------------------+-----------------------------------------------------------------------------------------------------------------+----------------------------------+
| GATEWAY_ALLOW_ORIGIN            | CORS configuration listing all domains allowed to use the gateway. Use "*" to allow all.                        | ""                               |
+---------------------------------+-----------------------------------------------------------------------------------------------------------------+----------------------------------+
| GATEWAY_REDIS_HOST              | The hostname/ip of the Redis instance used for persisting sessions.                                             | renku-gw-redis                   |
+---------------------------------+-----------------------------------------------------------------------------------------------------------------+----------------------------------+
| GITLAB_URL                      | The URL of the Gitlab instance to proxy.                                                                        | http://gitlab.renku.build        |
+---------------------------------+-----------------------------------------------------------------------------------------------------------------+----------------------------------+
| GITLAB_CLIENT_ID                | The client ID for the gateway in Gitlab.                                                                        | renku-ui                         |
+---------------------------------+-----------------------------------------------------------------------------------------------------------------+----------------------------------+
| GITLAB_CLIENT_SECRET            | The corresponding secret.                                                                                       | no-secret-needed                 |
+---------------------------------+-----------------------------------------------------------------------------------------------------------------+----------------------------------+
| JUPYTERHUB_URL                  | The URL of the JupyterHub.                                                                                      | {{HOST_NAME}}/jupyterhub         |
+---------------------------------+-----------------------------------------------------------------------------------------------------------------+----------------------------------+
| JUPYTERHUB_CLIENT_ID            | The client ID for the gateway in JupyterHub. This corresponds to the service oauth_client_id.                   | gateway                          |
+---------------------------------+-----------------------------------------------------------------------------------------------------------------+----------------------------------+
| JUPYTERHUB_CLIENT_SECRET        | The client secret for the gateway in JupyterHub. This corresponds to the service api_token.                     | dummy-secret                     |
+---------------------------------+-----------------------------------------------------------------------------------------------------------------+----------------------------------+
| KEYCLOAK_URL                    | The URL of the Keycloak instance.                                                                               | http://keycloak.renku.build:8080 |
+---------------------------------+-----------------------------------------------------------------------------------------------------------------+----------------------------------+
| OIDC_CLIENT_ID                  | The client ID for the gateway in Keycloak.                                                                      | gateway                          |
+---------------------------------+-----------------------------------------------------------------------------------------------------------------+----------------------------------+
| OIDC_CLIENT_SECRET              | The client secret for the gateway in Keycloak.                                                                  | dummy-secret                     |
+---------------------------------+-----------------------------------------------------------------------------------------------------------------+----------------------------------+
| GATEWAY_SERVICE_PREFIX          | The URL prefix for the gateway.                                                                                 | /                                |
+---------------------------------+-----------------------------------------------------------------------------------------------------------------+----------------------------------+
| GATEWAY_ENDPOINT_CONFIG_FILE    | The JSON definition of the API proxying endpoints.                                                              | endpoints.json                   |
+---------------------------------+-----------------------------------------------------------------------------------------------------------------+----------------------------------+
| CLI_CLIENT_ID                   | The client ID for the gateway's CLI client in Keycloak.                                                         | renku-cli                        |
+---------------------------------+-----------------------------------------------------------------------------------------------------------------+----------------------------------+
| CLI_CLIENT_SECRET               | The client secret for the gateway's CLI client in Keycloak.                                                     | dummy-secret                     |
+---------------------------------+-----------------------------------------------------------------------------------------------------------------+----------------------------------+
| CLI_LOGIN_TIMEOUT               | The validity of CLI login nonce in seconds.                                                                     | 300                              |
+---------------------------------+-----------------------------------------------------------------------------------------------------------------+----------------------------------+

Login workflow
--------------

To collect the user's token from the various backend services, the gateway uses the OAuth2/OIDC protocol and redirects the users to each of them.

.. image:: docs/login.png
  :width: 979


Redis storage
-------------

To allow server-side sessions, the gateway relies on Redis.

+------------------------------------------------------------+---------------------------------------------------------------------------------------------------------------------------+-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+
| key                                                        | value                                                                                                                     | remarks                                                                                                                                                                                                                                                     |
+============================================================+===========================================================================================================================+=============================================================================================================================================================================================================================================================+
| sessions_{{session key}}                                   | a dictionary with some temporary states (redirect_urls, login states, cli_token) and the user's Keycloak access token.    | The session key is managed by Flask-KVsession and kept in a secured, http-only cookie.                                                                                                                                                                      |
+------------------------------------------------------------+---------------------------------------------------------------------------------------------------------------------------+-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+
| cache_{{id sub}}_{{backend}}_{{token type}}                | The corresponding token                                                                                                   | Id sub is taken from the Keycloak access token in the session or Authorizazion header (after validation of the token). Current backends are Keycloak (kc), Gitlab (gl) and JupyterHub (jh). Token types can be access_token, refresh_token or id_token.     |
+------------------------------------------------------------+---------------------------------------------------------------------------------------------------------------------------+-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+

