Changes
=======

``v0.4.0``
----------
*(released TBD)*

This release uses Traefik for proxing requests to GitLab and JupyterHub.

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
