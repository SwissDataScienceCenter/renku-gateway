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

.. image:: https://pullreminders.com/badge.svg
    :target: https://pullreminders.com?ref=badge
    :alt: Pull reminders
    :align: right

==================
 Renku API gateway
==================

**The Renku platform is under very active development and should be considered highly
volatile.**

The Renku API gateway connects the different Renku clients to the various Renku backend
services (GitLab, Jupyterhub, etc). Currently, it mainly acts on the communication between
the Renku web UI and GitLab.


Quickstart
----------

In order to get an instance of Renku up and running, clone the main Renku
repository and follow these instructions_.

.. _instructions: https://renku.readthedocs.io/en/latest/developer/setup.html

Developing the gateway
----------------------
Once you have an instance of Renku running locally, you could modify the gateway code
and restart the platform through the :code:`make minikube-deploy` command to see the
changes. However, this will make for a very poor development experience as the deployment
process is optimized for production.

Instead we recommend connecting to your minikube (or any other kubernetes cluster) through
telepresence_. Once telepresence is installed, create a python environment and install
the necessary python dependencies by running :code:`pipenv install`. Then, start a
telepresence shell through :code:`make dev` and launch a development server by executing
the prompted command inside the telepresence shell.

.. _telepresence: https://www.telepresence.io/reference/install

The gateway in development setting is now available under the ip-address of your
minikube cluster (:code:`${minikube ip}/api`) and you should see requests from the
Renku UI appear in the logs.

So what is happening here? The command :code:`make dev` launches telepresence which
swaps the renku-gateway service in your minikube deployment for a locally running version of
the gateway served by a flask development server. This gives you live updates on code change
in a minikube deployment!

Running in a debugger
~~~~~~~~~~~~~~~~~~~~~

To run the gateway in the VS Code debugger, it is possible to use the *Python: Remote Attach*
launch configuration. The :code:`run-telepresence.sh` script prints the command to be used
for this purpose.

The prerequisite is that the :code:`ptvsd` module is installed in your Python environment.
This should be the case if you use the pipenv environment to run the gateway.

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

+---------------------------------+--------------------------------------------------------------------------------------------------------------+----------------------------------+
| Variable name                   | Description                                                                                                  | Default value                    |
+=================================+==============================================================================================================+==================================+
| HOST_NAME                       | The URL of this service.                                                                                     | http://gateway.renku.build       |
+---------------------------------+--------------------------------------------------------------------------------------------------------------+----------------------------------+
| GATEWAY_SECRET_KEY              | The key used, among others, to encrypt session cookies. This parameter is mandatory and there is no default. | -                                |
+---------------------------------+--------------------------------------------------------------------------------------------------------------+----------------------------------+
| GATEWAY_ALLOW_ORIGIN            | CORS configuration listing all domains allowed to use the gateway. Use "*" to allow all.                     | ""                               |
+---------------------------------+--------------------------------------------------------------------------------------------------------------+----------------------------------+
| GATEWAY_REDIS_HOST              | The hostname/ip of the Redis instance used for persisting sessions.                                          | renku-gw-redis                   |
+---------------------------------+--------------------------------------------------------------------------------------------------------------+----------------------------------+
| GITLAB_URL                      | The URL of the Gitlab instance to proxy.                                                                     | http://gitlab.renku.build        |
+---------------------------------+--------------------------------------------------------------------------------------------------------------+----------------------------------+
| GITLAB_CLIENT_ID                | The client ID for the gateway in Gitlab.                                                                     | renku-ui                         |
+---------------------------------+--------------------------------------------------------------------------------------------------------------+----------------------------------+
| GITLAB_CLIENT_SECRET            | The corresponding secret.                                                                                    | no-secret-needed                 |
+---------------------------------+--------------------------------------------------------------------------------------------------------------+----------------------------------+
| JUPYTERHUB_URL                  | The URL of the JupyterHub.                                                                                   | {{HOST_NAME}}/jupyterhub         |
+---------------------------------+--------------------------------------------------------------------------------------------------------------+----------------------------------+
| JUPYTERHUB_CLIENT_ID            | The client ID for the gateway in JupyterHub. This corresponds to the service oauth_client_id.                | gateway                          |
+---------------------------------+--------------------------------------------------------------------------------------------------------------+----------------------------------+
| JUPYTERHUB_CLIENT_SECRET        | The client secret for the gateway in JupyterHub. This corresponds to the service api_token.                  | dummy-secret                     |
+---------------------------------+--------------------------------------------------------------------------------------------------------------+----------------------------------+
| KEYCLOAK_URL                    | The URL of the Keycloak instance.                                                                            | http://keycloak.renku.build:8080 |
+---------------------------------+--------------------------------------------------------------------------------------------------------------+----------------------------------+
| OIDC_CLIENT_ID                  | The client ID for the gateway in Keycloak.                                                                   | gateway                          |
+---------------------------------+--------------------------------------------------------------------------------------------------------------+----------------------------------+
| OIDC_CLIENT_SECRET              | The client secret for the gateway in Keycloak.                                                               | dummy-secret                     |
+---------------------------------+--------------------------------------------------------------------------------------------------------------+----------------------------------+
| GATEWAY_SERVICE_PREFIX          | The URL prefix for the gateway.                                                                              | /                                |
+---------------------------------+--------------------------------------------------------------------------------------------------------------+----------------------------------+
| GATEWAY_ENDPOINT_CONFIG_FILE    | The JSON definition of the API proxying endpoints.                                                           | endpoints.json                   |
+---------------------------------+--------------------------------------------------------------------------------------------------------------+----------------------------------+

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

Extending the gateway
---------------------

If you want to add more services behind the gateway, you can easily configure the mapping in :code:`endpoints.json` (or point to another configuration file).

Adding a service backend handling authentication
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This part is still work in progress to make it plug and play. But the idea is to add the necessary http endpoints for the login/redirect/tokens for the external service and start the process by redirecting from the last service. (At the moment Keycloak -> Gitlab -> JupyterHub).
You can take as an example the :code:`gitlab_auth.py` or :code:`jupyterhub_auth.py` files and implement the :code:`/auth/<your service>/login`, :code:`/auth/<your service>/token` and :code:`/auth/<your service>/logout` endpoints.
You can then populate the Redis cache with the collected tokens that identify the user and can be used for authorization towards some API.

Adding an authorization method
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

If your backend API needs a specific authentication/authorization method you can write an auth processor, like the :code:`GitlabUserToken`, :code:`JupyterhubUserToken` or :code:`KeycloakAccessToken`.

Processing the requests and responses
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

By implementing a class extending the base processor, you can pre-process the incomming request and/or the returning response. You can have a look at the :code:`gitlab_processor.py` as a starting example.
