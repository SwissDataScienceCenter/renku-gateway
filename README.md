# Renku Gateway

[![Coverage Status](https://coveralls.io/repos/github/SwissDataScienceCenter/renku-gateway/badge.svg?branch=master)](https://coveralls.io/github/SwissDataScienceCenter/renku-gateway?branch=master)

An API Gateway and session service for Renku.

## Package dependencies and import chains

```mermaid
flowchart TB
    gwerrors -.-> config
    gwerrors --> models
    models --> config
    config --> other[every other package]
    models -.-> other
    gwerrors -.-> other
```

The arrows point in the direction in which a module is imported. So an arrow pointing from 
`models` to `config` means that models can import (depend on) config.

Restrictions:
- `gwerrors` can NOT import from any other package in the gateway
- `models` can only import from `gwerrors`
- `config` can only import from `gwerrors` or `models`
- the rest of the packages can import stuff from anywhere

Circular dependencies are still possible with the above setup but less likely because `config`, `gwerrors`
and `models` are the packages that are most commonly used by other packages. There is no linting
or any other checks or guards in place to enforce this. Hopefully a convention / agreement like this
is enough to avoid problems.

## Login server

The login routes handle authentication for web-based clients.

When a user starts the login flow, a new session is created which will reference authentication tokens once the login flow is done.

## Login flow

```mermaid
sequenceDiagram
    Participant GW
    Participant KC as Keycloak
    Participant GL as Gitlab
    Participant U as User
    Participant R as Renku
    
    U ->> GW: /login
    GW ->> GW: Start login sequence
    GW ->> KC: /authz?client_id=renku&redirect_uri=/callback&state=123random
    KC ->> KC: User enters credentials, approves access
    KC ->> GW: /callback?state=123random&code=secret123
    GW ->> GW: Validate state parameter
    GW -->> KC: [Exchange code for access token]
    GW ->> GW: Continue login sequence
    GW ->> GL: /authorize?client_id=renku&redirect_uri=/callback&state=456random
    GL ->> GL: User enters credentials, approves access
    GL ->> GW: /callback?state=456random&code=secretxyz
    GW ->> GW: Validate state parameter
    GW -->> GL: [Exchange code for access token]
    GW ->> GW: Clear up login state
    GW ->> R: Finally navigate to Renku    
```

## Reverse proxy

The reverse proxy routes incoming requests to the appropriate service and injects the corresponding credentials.
