Changes
=======

`0.9.4 <https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.9.3...0.9.4>`__ (2021-03-10)
------------------------------------------------------------------------------------------------------

Features
~~~~~~~~

-  add routing for GitLab authenticated Knowledge Graph requests
   (`#382 <https://github.com/SwissDataScienceCenter/renku-gateway/issues/382>`__)
   (`eed159f <https://github.com/SwissDataScienceCenter/renku-gateway/commit/eed159fac4e104adb7bdf6551c9ee82acf5aefba>`__)

`0.9.3 <https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.9.2...0.9.3>`__ (2020-11-30)
------------------------------------------------------------------------------------------------------

Features
~~~~~~~~

-  **auth:** pass on KC id token to core service
   (`#299 <https://github.com/SwissDataScienceCenter/renku-gateway/issues/299>`__)
   (`3d34d26 <https://github.com/SwissDataScienceCenter/renku-gateway/commit/3d34d26b38a87ec7cc5e5125286144b7c212f1b8>`__)


`0.9.2 <https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.9.1...0.9.2>`__ (2020-10-28)
------------------------------------------------------------------------------------------------------

Bug Fixes
~~~~~~~~~

-  **app:** fix JupyterHub logout and logout redirection
   (`a7ffbed <https://github.com/SwissDataScienceCenter/renku-gateway/commit/a7ffbed>`__)


`0.9.1 <https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.9.0...0.9.1>`__ (2020-10-06)
------------------------------------------------------------------------------------------------------

Bug Fixes
~~~~~~~~~

-  **charts:** fix a bug in the direct routing to gitlab
   (`4fc0da6 <https://github.com/SwissDataScienceCenter/renku-gateway/commit/4fc0da62c96a9426aa8e85569e3678cd4f3540c0>`__)
-  adjust the time buffer in the token expiration date check function
   (`3048aee <https://github.com/SwissDataScienceCenter/renku-gateway/commit/3048aeebddc2e3319a39a74524a00ec8e32bac0d>`__)


`0.9.0 <https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.8.0...0.9.0>`__ (2020-08-11)
------------------------------------------------------------------------------------------------------

Features
~~~~~~~~

- enable kubernetes versions > 1.15
   (`b226e47 <https://github.com/SwissDataScienceCenter/renku-gateway/commit/b226e4720dac52d031e5ebe991cb1c1749ee0e39>`__)

Bug Fixes
~~~~~~~~~

-  avoid crash when invoking the core-service for a user with non-latin-1 characters in their name
   (`#253 <https://github.com/SwissDataScienceCenter/renku-gateway/issues/253>`__)
   (`6894ca3 <https://github.com/SwissDataScienceCenter/renku-gateway/commit/6894ca368a9a166290e927260e3d92c34cb9acb9>`__)
-  correct token swapping for core service
   (`b9b1cd1 <https://github.com/SwissDataScienceCenter/renku-gateway/commit/b9b1cd11e1e3787a01c84c35363a617b8dc76c6b>`__)

BREAKING CHANGES
~~~~~~~~~~~~~~~~

- kubernetes versions < 1.14 are not supported anymore


`0.8.0 <https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.7.1...0.8.0>`__ (2020-05-26)
------------------------------------------------------------------------------------------------------

Code Refactoring
~~~~~~~~~~~~~~~~

- **black:** apply black formatting test it on future PRs
  (`956c767 <https://github.com/SwissDataScienceCenter/renku-gateway/commit/956c767733c75587c1d55171d387041be88774a7>`__).
- **dependabot:** python dependencies were updated and dependabot enabled
  (`4bfc0b1 <https://github.com/SwissDataScienceCenter/renku-gateway/commit/4bfc0b1c67c5f7f959893e77462e1b65a42c1b5d>`__).
- **GitLab:** Adapt to new GitLab logout behaviour
  (`01dff94 <https://github.com/SwissDataScienceCenter/renku-gateway/commit/01dff9478f5a2fdd1785a1926380819904585e25>`__).

BREAKING CHANGES
~~~~~~~~~~~~~~~~

* **GitLab version:** We now assume a GitLab version `>= 12.9.0` per default. When deploying Renku
  through the official helm chart, no changes to the deployment `values.yaml` file are necessary as
  we also bump the GitLab version in the same
  (`#1118 <https://github.com/SwissDataScienceCenter/renku/pull/1118)>`__).
  GitLab versions `< 12.7.0` can be used with this version too, but a ``.Values.oldGitLabLogout: true``
  has to be set explicitly.


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
