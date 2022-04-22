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

The Renku API gateway connects the different Renku clients to the
various Renku backend services (GitLab, Renku components etc). It
consists of three logically distinct parts:

-  A traefik reverse-proxy which receives *all* incoming API requests,
   authenticates requests using an external service as `forward-auth
   middleware`_ before forwarding requests to the dedicated backend 
   service for the requested resource.
-  A flask service which acts as the forward-auth middleware mentioned
   above. It acts only on the request headers which it receives from
   traefik, where it expects to find a keycloak access token as bearer
   token in the authorization header. It then swaps it for the right
   headers which will allow the backend service to properly authenticate
   the user and/or check if the user is authorized to access the requested
   resource.
-  A flask service which the browser can be redirected to by clients in
   order to acquire access- and potentially refresh- and id-tokens
   tokens for the backend services (ie currently Keycloak and Gitlab).

Note that currently, although they have distinct roles, the two flask
services are deployed together as one flask application which share 
some libraries and utility functions for handling access tokens and 
storing them in redis.

.. _forward-auth middleware: https://doc.traefik.io/traefik/middlewares/http/forwardauth/


Developing the gateway-auth component
-------------------------------------
The renku gateway-auth component is best developped in the context of a full renku 
deployment. In order to get an instance of Renku up and running, clone the main Renku
repository and follow these instructions_.

.. _instructions: https://renku.readthedocs.io/en/latest/developer/setup.html

Once you have an instance of Renku running, you could modify the gateway code, build the 
image, re-build the chart, redeploy, etc... This will make for a poor development experience
with very long feedback cycles.

Instead, we recommend intercepting traffic to the gateway-auth component and routing it to
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

You can run a test suite (very incomplete, mostly test boilerplate) with

::

    $ pipenv run pytest

Configuration
-------------
The standard way of deploying the renku-gateay component is through the helm chart 
which is part of this repository. Check out the the `values file`_ for explanations 
of the various settings and defaults.

.. _values: helm-chart/renku-gateway/values.yaml


Login flow
----------

To collect the user's token from the various backend services, the gateway uses the 
OAuth2/OIDC protocol and redirects the users to each of them. Check out this diagram_
for details.

.. _diagram: app/web/login-flow.md


Redis storage
-------------

The gateway relies on Redis for storing the users access tokens to Keycloak and Gitlab.
More precicely, for each user and each OAuth2 provider, we store a serialized instance 
of the :code:`RenkuWebApplicationClient` class defined here_, which contain the users 
access token for the provider, and - if applicable - refresh- and id-token.

.. _here: app/common/oauth_client.py#L33
