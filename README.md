# Renku Gateway

A reverse proxy and Oauth2 flows handler for Renku.

## Package dependencies and import chains

```mermaid
flowchart TB
    gwerrors -.-> config
    gwerrors --> models
    models --> config
    config --> other[every other package]
    models -.-> other
    gwerrors -.-> othermermaid
```

The arrows point in the direction in which a module is imported. So an arrow pointing from 
`models` to `config` means that models can import (depend on) confing.

Restrictions:
- `gwerrors` can NOT import from any other package in the gateway
- `models` can only import from `gwerrors`
- `config` can only import from `gwerrors` or `models`
- the rest of the packages can import stuff from anywhere

Circular dependencies are still possible with the above setup but less likely because `config`, `gwerrors`
and `models` are the packages that are most commonly used by other packages. There is no linting
or any other checks or guards in place to enforce this. Hopefully a convention / agreement like this
is enough to avoid problems.

## Oauth2 flows

```mermaid
sequenceDiagram
    Participant GW
    Participant KC as Keycloak
    Participant GL as Gitlab
    Participant U as User
    Participant R as Renku
    
    U ->> GW: /login
    GW ->> GW: Generate state parameters for all providers
    GW ->> KC: /authz?client_id=renku&redirect_uri=/callback&state=123random
    KC ->> KC: User enters credentials, approves access
    KC ->> GW: /callback?state=123radnom&code=secret123
    GW ->> GW: Validate state parameter
    GW -->> KC: [Exchange code for access token]
    GW ->> GL: /authorize?client_id=renku&redirect_uri=/callback&state=456random
    GL ->> GL: User enters credentials, approves access
    GL ->> GW: /callback?state=456random&code=secretxyz
    GW ->> GW: Validate state parameter
    GW -->> GL: [Exchange code for access token]
    GW ->> R: Finally navigate to Renku    
```

## Device login flow

```mermaid
sequenceDiagram
    Participant GW
    Participant KC as Keycloak
    Participant GL as Gitlab
    Participant U as User
    Participant CLI
    
    U ->> CLI: renku login
    CLI ->> GW: /api/auth/device/login/session
    GW ->> GW: Generate state parameters for all providers, generate session
    GW ->> KC: Start device flow<br>POST /auth/realms/Renku/protocol/openid-connect/auth/device
    GW ->> CLI: Return message to go to /api/auth/device/login?session=<session-id>
    CLI ->> U: Show message to user
    CLI -->> KC: Keep querying KC for device token [in background]
    CLI -->> KC: 
    CLI -->> KC: 
    U ->> GW: /api/auth/device/login?session=<session-id>
    GW ->> GW: Load session, pass on to oAuthNext
    GW ->> GL: /authorize?client_id=renku&redirect_uri=/callback&state=456random
    GL ->> GL: User enters credentials, approves access
    GL ->> GW: /callback?state=456random&code=secretxyz
    GW ->> GW: Validate state parameter
    GW -->> GL: [Exchange code for access token]
    GW ->> KC: Redirect to proper url to finalize device flow
    KC ->> KC: Enter credentials, approve access
    CLI ->> KC: Finally acquire credentials
    CLI ->> GW: save credentials POST /api/auth/device/login/token (pass session ID as cookie or header)
```

