# Changes

## [0.19.2](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.19.1...0.19.2) (2023-04-20)

### Bug Fixes

* **chart:** HPA for reverse proxy ([#643](https://github.com/SwissDataScienceCenter/renku-gateway/issues/643)) ([de11c52](https://github.com/SwissDataScienceCenter/renku-gateway/commit/de11c52c0e24d4c7f3aaeb52a8d75a6782ee74ea))



## [0.19.1](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.19.0...0.19.1) (2023-04-13)


### Bug Fixes

* /api/repos path should be just /repos ([#640](https://github.com/SwissDataScienceCenter/renku-gateway/issues/640)) ([4d966ce](https://github.com/SwissDataScienceCenter/renku-gateway/commit/4d966ce986c3281459e64fcfaf2f03608baf86a2))




## [0.19.0](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.18.1...0.19.0) (2023-03-31)


### Features

* **app:** sticky sessions middleware ([#630](https://github.com/SwissDataScienceCenter/renku-gateway/issues/630)) ([06ff27c](https://github.com/SwissDataScienceCenter/renku-gateway/commit/06ff27cbdc7ba7f5bc7cfbf235c6e643042faecd))
* **app:** use golang echo as reverse proxy ([#623](https://github.com/SwissDataScienceCenter/renku-gateway/issues/623)) ([58e3cd0](https://github.com/SwissDataScienceCenter/renku-gateway/commit/58e3cd06b6da46cfd5f1d8ec929fee7db1873224))



## [0.18.1](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.18.0...0.18.1) (2023-02-24)

### Bug Fixes

* use offline access tokens for renku client ([#632](https://github.com/SwissDataScienceCenter/renku-gateway/issues/632)) ([dc93620](https://github.com/SwissDataScienceCenter/renku-gateway/commit/dc93620108f020ba950029de64213169869ed619))



## [0.18.0](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.17.0...0.18.0) (2022-12-12)

### Bug Fixes

* **app:** do not remove redis clients on logout ([#616](https://github.com/SwissDataScienceCenter/renku-gateway/issues/616)) ([8ca7fc9](https://github.com/SwissDataScienceCenter/renku-gateway/commit/8ca7fc986aab3d9d5b7305d4eaa8dbcd37f1a2bb))
* **app:** snyk vulnerabilities ([#615](https://github.com/SwissDataScienceCenter/renku-gateway/issues/615)) ([77616b7](https://github.com/SwissDataScienceCenter/renku-gateway/commit/77616b7fdcd112d2fb8c759e8f4663eea0ca7222))


### Features

* **app:** add endpoint for refreshing expired gitlab tokens ([#613](https://github.com/SwissDataScienceCenter/renku-gateway/issues/613)) ([8d0c2eb](https://github.com/SwissDataScienceCenter/renku-gateway/commit/8d0c2eb5df76a54170132e1bdcefc281c1709530))


## [0.17.0](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.16.0...0.17.0) (2022-11-29)

### Bug Fixes

* remove trailing slash from redirect links ([#598](https://github.com/SwissDataScienceCenter/renku-gateway/issues/598)) ([024b5f5](https://github.com/SwissDataScienceCenter/renku-gateway/commit/024b5f542cc03e782216412a69563b5b032ec6b8))

### Features

* adopt renku styles in logout pages ([#521](https://github.com/SwissDataScienceCenter/renku-gateway/issues/521), [#601](https://github.com/SwissDataScienceCenter/renku-gateway/issues/601)) ([20404fb](https://github.com/SwissDataScienceCenter/renku-gateway/commit/20404fbb7b8e6e67cffb3b6ad1318e9a88e47d32))


## [0.16.0](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.15.0...0.16.0) (2022-10-24)

### Features

* redirect /gitlab to external GitLab url ([#596](https://github.com/SwissDataScienceCenter/renku-gateway/issues/596)) ([5bf0701](https://github.com/SwissDataScienceCenter/renku-gateway/commit/5bf0701c54ca730b3b2cabc1a98c7b2efca33ace)), closes [SwissDataScienceCenter/renku#2741](https://github.com/SwissDataScienceCenter/renku/issues/2741)


## [0.15.0](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.14.0...0.15.0) (2022-10-11)

### Bug Fixes

* **app:** re-initialize keycloak client if needed ([#590](https://github.com/SwissDataScienceCenter/renku-gateway/issues/590)) ([fc69fb5](https://github.com/SwissDataScienceCenter/renku-gateway/commit/fc69fb54d979ae69f31ea4de34e240a9fb79de45))
* **ci:** docker image build ([#581](https://github.com/SwissDataScienceCenter/renku-gateway/issues/581)) ([e3158b1](https://github.com/SwissDataScienceCenter/renku-gateway/commit/e3158b12bff0763c75566b6f4c19f2b1227f61eb))

### Features

* remove anon-id cookies creation ([#584](https://github.com/SwissDataScienceCenter/renku-gateway/issues/584)) ([122eb05](https://github.com/SwissDataScienceCenter/renku-gateway/commit/122eb0572fc5d3a41799dadff1aa6d5d3685430b)), closes [SwissDataScienceCenter/renku-ui#1601](https://github.com/SwissDataScienceCenter/renku-ui/issues/1601)


## [0.14.0](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.13.1...0.14.0) (2022-06-24)

### Features

- **chart**: use official traefik helm chart ([#561](https://github.com/SwissDataScienceCenter/renku-gateway/issues/561))
    ([3d48f66](https://github.com/SwissDataScienceCenter/renku-gateway/commit/3d48f66c5c4aa0ca7a148ea504849c4b908badd0))
- **app**: refactor traefik rules ([#561](https://github.com/SwissDataScienceCenter/renku-gateway/issues/561))
    ([9685360](https://github.com/SwissDataScienceCenter/renku-gateway/commit/96853609707c4c2468220613ac49ec21be39fcd5))

## [0.13.1](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.13.0...0.13.1) (2022-03-16)

### Bug Fixes

-   **chart:** Fix Cross-Origin Resource Sharing allowed origin list.
    The parameter is now under `gateway.allowOrigin`.
    ([#554](https://github.com/SwissDataScienceCenter/renku-gateway/issues/554))
    ([527877c](https://github.com/SwissDataScienceCenter/renku-gateway/commit/527877c309d535f50df97cf83963bb63549ff0fc))

## [0.13.0](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.12.3...0.13.0) (2022-02-25)

### Bug Fixes

-   **chart:** modify values for global redis
    ([#552](https://github.com/SwissDataScienceCenter/renku-gateway/issues/552))
    ([3b5fdff](https://github.com/SwissDataScienceCenter/renku-gateway/commit/3b5fdffcd883cbe5af4566558b32593b68f9fb2e))

### BREAKING CHANGES

\- This version does not come with its own Redis instance. Instead it
relies on a global instance provided elsewhere. When the gateway is
deployed as part of Renku by default this global Redis is provided by
the Renku Helm chart. However, due to similar changes made in the global
Renku chart, this version of the gateway is only compatible with Renku
versions after Renku 0.12.1 (excluding 0.12.1 itself). The breaking
changes are in the organization and fields under
`global.redis` in the `values.yaml` file for the
Helm chart.

## [0.12.3](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.12.2...0.12.3) (2022-02-15)

### Bug Fixes

-   update sentry values
    ([#533](https://github.com/SwissDataScienceCenter/renku-gateway/issues/533))
    ([0734e8e](https://github.com/SwissDataScienceCenter/renku-gateway/commit/0734e8ef82b913d0744e0a5915433eaa27791607))

## [0.12.2](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.12.1...0.12.2) (2022-02-10)

### Bug Fixes

-   **chart:** errors in helm templating for redis instance
    ([#544](https://github.com/SwissDataScienceCenter/renku-gateway/issues/544))
    ([7a23a2f]((https://github.com/SwissDataScienceCenter/renku-gateway/commit/7a23a2fc8920ad5f39771c6961ec8bc428515d08))

## [0.12.1](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.12.0...0.12.1) (2022-02-09)

### Bug Fixes

-   **chart:** fully adapt to global redis
    ([#537](https://github.com/SwissDataScienceCenter/renku-gateway/issues/537))
    ([9003029]((https://github.com/SwissDataScienceCenter/renku-gateway/commit/90030292fda4e65787cbfd3f1e600f625d1b11f5))

## [0.12.0](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.11.0...0.12.0) (2022-02-08)

### Features

-   modify for custom CA certificates
    ([#486](https://github.com/SwissDataScienceCenter/renku-gateway/issues/486))
    ([c6774a4]((https://github.com/SwissDataScienceCenter/renku-gateway/commit/c6774a421753e15bf2aabe73a66518c08240c7b4))

## [0.11.0](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.10.2...0.11.0) (2021-12-08)

### Features

-   use global redis instance
    ([42c8e9e](https://github.com/SwissDataScienceCenter/renku-gateway/commit/42c8e9edc5ea1ae85e2455268b5d274e25f0f214))

### Bug Fixes

-   remove path constraint on anon-id cookie
    ([#510](https://github.com/SwissDataScienceCenter/renku-gateway/issues/510))
    ([b5c662c](https://github.com/SwissDataScienceCenter/renku-gateway/commit/b5c662c72b667b7dc9431559f2648241c0feb03e))

## [0.10.2](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.10.1...0.10.2) (2021-11-23)

### Features

-   add gitlab graphql as a separate route
    ([#491](https://github.com/SwissDataScienceCenter/renku-gateway/issues/491))
    ([7cd80f3](https://github.com/SwissDataScienceCenter/renku-gateway/commit/7cd80f38d9e674787a5f88588f5b3ff605fbaca9))

## [0.10.1](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.10.0...0.10.1) (2021-10-14)

### Bug Fixes

-   **auth:** prevent exception when using Keycloak access tokens
    ([2abd0cb](https://github.com/SwissDataScienceCenter/renku-gateway/commit/2abd0cba3f3e4b3426c7744dd9ecceca43e01454))
-   **auth:** log out from GitLab upon Renku logout
    ([da0897d](https://github.com/SwissDataScienceCenter/renku-gateway/commit/da0897d42d26e38abbf6fcb288dbf06efc2bca33))

## [0.10.0](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.9.5...0.10.0) (2021-09-01)

### Features

-   enable Amalthea-based sessions
    ([ec87dc6](https://github.com/SwissDataScienceCenter/renku-gateway/commit/ec87dc6f679d17d7504729478fd0c18dc9d12c91))

### BREAKING CHANGES

-   This version will not work with older versions of renku-notebooks,
    the intended combinations of component versions can be found in the
    main Renku chart.

## [0.9.5](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.9.4...0.9.5) (2021-07-21)

### Features

-   add renku headers to nb service requests
    ([754a97f](https://github.com/SwissDataScienceCenter/renku-gateway/commit/754a97fe9a82effc9544c10f034aa815e35a8a3a))
-   authenticate gitlab requests
    ([#419](https://github.com/SwissDataScienceCenter/renku-gateway/issues/419))
    ([efd39bb](https://github.com/SwissDataScienceCenter/renku-gateway/commit/efd39bbcbe51f87984735fd0c15b51acfb56ac7c))
-   support for CLI login
    ([#367](https://github.com/SwissDataScienceCenter/renku-gateway/issues/367))
    ([8d97690](https://github.com/SwissDataScienceCenter/renku-gateway/commit/8d97690f879a7def6dd8310324616f3eabdb62d0))

## [0.9.4](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.9.3...0.9.4) (2021-03-10)

### Features

-   add routing for GitLab authenticated Knowledge Graph requests
    ([#382](https://github.com/SwissDataScienceCenter/renku-gateway/issues/382))
    ([eed159f](https://github.com/SwissDataScienceCenter/renku-gateway/commit/eed159fac4e104adb7bdf6551c9ee82acf5aefba))

## [0.9.3](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.9.2...0.9.3) (2020-11-30)

### Features

-   **auth:** pass on KC id token to core service
    ([#299](https://github.com/SwissDataScienceCenter/renku-gateway/issues/299))
    ([3d34d26](https://github.com/SwissDataScienceCenter/renku-gateway/commit/3d34d26b38a87ec7cc5e5125286144b7c212f1b8))

## [0.9.2](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.9.1...0.9.2) (2020-10-28)

### Bug Fixes

-   **app:** fix JupyterHub logout and logout redirection
    ([a7ffbed](https://github.com/SwissDataScienceCenter/renku-gateway/commit/a7ffbed))

## [0.9.1](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.9.0...0.9.1) (2020-10-06)

### Bug Fixes

-   **charts:** fix a bug in the direct routing to gitlab
    ([4fc0da6](https://github.com/SwissDataScienceCenter/renku-gateway/commit/4fc0da62c96a9426aa8e85569e3678cd4f3540c0))
-   adjust the time buffer in the token expiration date check function
    ([3048aee](https://github.com/SwissDataScienceCenter/renku-gateway/commit/3048aeebddc2e3319a39a74524a00ec8e32bac0d))

## [0.9.0](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.8.0...0.9.0) (2020-08-11)

### Features

-   enable kubernetes versions \> 1.15 ([b226e47](https://github.com/SwissDataScienceCenter/renku-gateway/commit/b226e4720dac52d031e5ebe991cb1c1749ee0e39))

### Bug Fixes

-   avoid crash when invoking the core-service for a user with
    non-latin-1 characters in their name
    ([#253](https://github.com/SwissDataScienceCenter/renku-gateway/issues/253))
    ([6894ca3](https://github.com/SwissDataScienceCenter/renku-gateway/commit/6894ca368a9a166290e927260e3d92c34cb9acb9))
-   correct token swapping for core service
    ([b9b1cd1](https://github.com/SwissDataScienceCenter/renku-gateway/commit/b9b1cd11e1e3787a01c84c35363a617b8dc76c6b))

### BREAKING CHANGES

-   kubernetes versions \< 1.14 are not supported anymore

## [0.8.0](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.7.1...0.8.0) (2020-05-26)

### Code Refactoring

-   **black:** apply black formatting test it on future PRs
    ([956c767](https://github.com/SwissDataScienceCenter/renku-gateway/commit/956c767733c75587c1d55171d387041be88774a7)).
-   **dependabot:** python dependencies were updated and dependabot
    enabled
    ([4bfc0b1](https://github.com/SwissDataScienceCenter/renku-gateway/commit/4bfc0b1c67c5f7f959893e77462e1b65a42c1b5d)).
-   **GitLab:** Adapt to new GitLab logout behaviour
    ([01dff94](https://github.com/SwissDataScienceCenter/renku-gateway/commit/01dff9478f5a2fdd1785a1926380819904585e25)).

### BREAKING CHANGES

-   **GitLab version:** We now assume a GitLab version
    `>=12.9.0` per default. When deploying Renku through the
    official helm chart, no changes to the deployment
    `values.yaml` file are necessary as we also bump the
    GitLab version in the same
    ([#1118](https://github.com/SwissDataScienceCenter/renku/pull/1118))).
    GitLab versions `< 12.7.0` can be used with this
    version too, but a `.Values.oldGitLabLogout: true` has to be set
    explicitly.

## [0.7.1](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.7.0...0.7.1) (2020-04-24)

### Features

-   add routing and \"login\" route for anonymous notebooks sessions.
    ([#193](https://github.com/SwissDataScienceCenter/renku-gateway/issues/193)
    ,
    [#195](https://github.com/SwissDataScienceCenter/renku-gateway/issues/195)),

## [0.7.0](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.6.0...0.7.0) (2020-03-05)

### Features

-   add core service routing
    ([#181](https://github.com/SwissDataScienceCenter/renku-gateway/issues/181)
    ,
    [ee94e63](https://github.com/SwissDataScienceCenter/renku-gateway/commit/ee94e63bab0d3e70cf2cdc23f12df1faf50c9592)),

## [0.6.0](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.5.0...0.6.0) (2019-11-04)

### Code Refactoring

-   **chart:** Several small changes to the charts, including the
    renaming of the main keycloak client application to be used from
    `gateway` to `renku`.
    ([b332cdc](https://github.com/SwissDataScienceCenter/renku-gateway/commit/b332cdc))

### Features

-   add a user profile endpoint which redirects the browser to Keycloak
    ([76a57bc](https://github.com/SwissDataScienceCenter/renku-gateway/commit/76a57bc)),
    closes
    [#173](https://github.com/SwissDataScienceCenter/renku-gateway/issues/173)

### BREAKING CHANGES

-   **chart:** Several small changes to the charts require corresponding
    changes in the Renku umbrella chart.

## [0.5.0](https://github.com/SwissDataScienceCenter/renku-gateway/compare/0.4.1...0.5.0) (2019-08-06)

### Bug Fixes

-   **traefik:** update graphql load balancer path
    ([4e1389f](https://github.com/SwissDataScienceCenter/renku-gateway/commit/4e1389f)),
    closes
    [#158](https://github.com/SwissDataScienceCenter/renku-gateway/issues/158)

### Code Refactoring

-   **graph:** remove legacy graph code
    ([1b7b9b2](https://github.com/SwissDataScienceCenter/renku-gateway/commit/1b7b9b2))

### Features

-   add graphql routing in traefik
    ([7a0271d](https://github.com/SwissDataScienceCenter/renku-gateway/commit/7a0271d)),
    closes
    [#158](https://github.com/SwissDataScienceCenter/renku-gateway/issues/158)

### BREAKING CHANGES

-   **graph:** graph API has been moved to another repo
    <https://github.com/SwissDataScienceCenter/renku-graph/tree/master/knowledge-graph>

## `v0.4.1`

*(released 2019-07-23)*

-   Remove restrictive rate limits for the notebooks service (
    [#155](https://github.com/SwissDataScienceCenter/renku-gateway/pull/155))
-   Use basic authentication with Jena (
    [#156](https://github.com/SwissDataScienceCenter/renku-gateway/issues/156),
    [#157](https://github.com/SwissDataScienceCenter/renku-gateway/pull/157))
-   Update SPARQL query for the Knowledge Graph endpoint (
    [#160](https://github.com/SwissDataScienceCenter/renku-gateway/issues/160),
    [#161](https://github.com/SwissDataScienceCenter/renku-gateway/pull/161))

## `v0.4.0`

*(released 2019-05-23)*

This release uses Traefik for proxying requests to GitLab and
JupyterHub.

## `v0.3.1`

*(released 2019-02-28)*

This release fixes an error in the implementation which prevented the
forwarded requests from being handled asynchronously. This alleviates
some of the observed performance issues related to the gateway.

## `v0.3.0`

*(released 2018-11-26)*

The most notable change is the use of a storage backend to support
stateful sessions. Namely a Redis instance is now spawned to store the
current user\'s session and the mapping to the backend API tokens.

* Redis is added to the helm dependencies (and its name overriden to avoid
  conflicts). New values can control its parameters, such as replication.

* GitLab and JupyterHub are added as OAuth2 providers, a service/application
  has to be registered into them to allow the gateway to proceed. The
  corresponding values are mandatory.

* Authentication of API calls on the gateway can be done with Keycloak access
  or refresh tokens, or a session cookie. The secret key for 
  encrypting cookies is a mandatory value with no defaults.

* Plug and play extensibility provided by decoupling the authentication /
  authorization and the query mapping. It is possible to add more
  backend APIs by extending one or two classes and controling the
  mapping in a configuration file: endpoints.json

## `v0.2.0`

*(released 2018-09-25)*

Initial release as a part of the larger Renku release. The gateway acts
as a stateless proxy between the UI and Gitlab, providing the necessary
endpoints for OAuth2/OpenID-connect login/logout and token retrieval.
Calls to GitLab are transformed to use a \"sudo token\" and taking the
identity of the user obtained from the Keycloak access token sent from
the UI.
