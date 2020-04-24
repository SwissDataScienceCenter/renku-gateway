Changes
=======

`0.7.1 <https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.7.0...0.7.1>`__ (2020-04-24)
------------------------------------------------------------------------------------------------------

Features
~~~~~~~~

-  add routing and "login" route for anonymous notebooks sessions.
   (`#193 <https://github.com/SwissDataScienceCenter/renku-gateway/issues/193>`__ ,
   `#195 <https://github.com/SwissDataScienceCenter/renku-gateway/issues/195>`__),


`0.7.0 <https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.6.0...0.7.0>`__ (2020-03-05)
------------------------------------------------------------------------------------------------------

Features
~~~~~~~~

-  add core service routing
   (`#181 <https://github.com/SwissDataScienceCenter/renku-gateway/issues/181>`__ ,
   `ee94e63 <https://github.com/SwissDataScienceCenter/renku-gateway/commit/ee94e63bab0d3e70cf2cdc23f12df1faf50c9592>`__),


`0.6.0 <https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.5.0...0.6.0>`__ (2019-11-04)
------------------------------------------------------------------------------------------------------


Code Refactoring
~~~~~~~~~~~~~~~~

-  **chart:** Several small changes to the charts, including the renaming of the main keycloak client
   application to be used from `gateway` to `renku`.
   (`b332cdc <https://github.com/SwissDataScienceCenter/renku-gateway/commit/b332cdc>`__)

Features
~~~~~~~~

-  add a user profile endpoint which redirects the browser to Keycloak
   (`76a57bc <https://github.com/SwissDataScienceCenter/renku-gateway/commit/76a57bc>`__),
   closes
   `#173 <https://github.com/SwissDataScienceCenter/renku-gateway/issues/173>`__

BREAKING CHANGES
~~~~~~~~~~~~~~~~

-  **chart:** Several small changes to the charts require corresponding changes in the Renku umbrella chart.

`0.5.0 <https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.4.1...0.5.0>`__ (2019-08-06)
------------------------------------------------------------------------------------------------------

Bug Fixes
~~~~~~~~~

-  **traefik:** update graphql load balancer path
   (`4e1389f <https://github.com/SwissDataScienceCenter/renku-gateway/commit/4e1389f>`__),
   closes
   `#158 <https://github.com/SwissDataScienceCenter/renku-gateway/issues/158>`__

Code Refactoring
~~~~~~~~~~~~~~~~

-  **graph:** remove legacy graph code
   (`1b7b9b2 <https://github.com/SwissDataScienceCenter/renku-gateway/commit/1b7b9b2>`__)

Features
~~~~~~~~

-  add graphql routing in traefik
   (`7a0271d <https://github.com/SwissDataScienceCenter/renku-gateway/commit/7a0271d>`__),
   closes
   `#158 <https://github.com/SwissDataScienceCenter/renku-gateway/issues/158>`__

BREAKING CHANGES
~~~~~~~~~~~~~~~~

-  **graph:** graph API has been moved to another repo
   https://github.com/SwissDataScienceCenter/renku-graph/tree/master/knowledge-graph


``v0.4.1``
----------
*(released 2019-07-23)*

* Remove restrictive rate limits for the notebooks service (
  `#155 <https://github.com/SwissDataScienceCenter/renku-gateway/pull/155>`_)
* Use basic authentication with Jena (
  `#156 <https://github.com/SwissDataScienceCenter/renku-gateway/issues/156>`_,
  `#157 <https://github.com/SwissDataScienceCenter/renku-gateway/pull/157>`_)
* Update SPARQL query for the Knowledge Graph endpoint (
  `#160 <https://github.com/SwissDataScienceCenter/renku-gateway/issues/160>`_,
  `#161 <https://github.com/SwissDataScienceCenter/renku-gateway/pull/161>`_)

``v0.4.0``
----------
*(released 2019-05-23)*

This release uses Traefik for proxying requests to GitLab and JupyterHub.

``v0.3.1``
----------
*(released 2019-02-28)*

This release fixes an error in the implementation which prevented the forwarded
requests from being handled asynchronously. This alleviates some of the observed
performance issues related to the gateway.


``v0.3.0``
----------
*(released 2018-11-26)*

The most notable change is the use of a storage backend to support stateful
sessions. Namely a Redis instance is now spawned to store the current user's
session and the mapping to the backend API tokens.

* Redis is added to the helm dependencies (and its name overriden to avoid
 conflicts). New values can control its parameters, such as replication.

* GitLab and JupyterHub are added as OAuth2 providers, a service/application
 has to be registered into them to allow the gateway to proceed. The
 corresponding values are mandatory.

* Authentication of API calls on the gateway can be done with Keycloak access
 or refresh tokens, or a session cookie. The secret key for encrypting cookies
 is a mandatory value with no defaults.

* Plug and play extensibility provided by decoupling the authentication /
 authorization and the query mapping. It is possible to add more backend APIs
 by extending one or two classes and controling the mapping in a configuration
 file: endpoints.json


``v0.2.0``
----------
*(released 2018-09-25)*

Initial release as a part of the larger Renku release. The gateway acts as a
stateless proxy between the UI and Gitlab, providing the necessary endpoints
for OAuth2/OpenID-connect login/logout and token retrieval.
Calls to GitLab are transformed to use a "sudo token" and taking the identity
of the user obtained from the Keycloak access token sent from the UI.
