## Login flow

Check out the login flow below in the diagram below. Note the following aspects:

- We use regular client-side sessions to store some short-lived state (temporary id, step
  in the login sequence, URL to redirect to on completion of the login sequence) in a 
  signed cookie.
- OAuth2 authorization codes and the retrieved access-, refresh- and id tokens are stored
  in redis as part of the serialized `RenkuWebApplicationClient`.
- The redis keys are formed from the users keycloak id (ie the `sub` claim). Since this 
  id is not known as we initiate the login process, we generate a random short-lived id
  for a login session which is replaced by the actual keycloak id as soon as this is known.

```mermaid
sequenceDiagram
    participant User
    participant Gateway
    participant Keycloak
    participant GitLab
    participant Redis

    User->>Gateway: /api/auth/login?redirect_url
    activate User
    activate Gateway
    Gateway-->>User: redirect
    deactivate Gateway

    User->>Keycloak: /auth/realms/Renku/protocol/openid-connect/auth
    activate Keycloak
    Keycloak-->>Keycloak: login page if needed
    Keycloak-->>User: redirect
    deactivate Keycloak
    
    User->>Gateway: /api/auth/token
    activate Gateway
    Redis->>Gateway: get Keycloak RenkuWebApplicationClient instance
    Gateway->>Keycloak: /auth/realms/Renku/<br>protocol/openid-connect/token
    activate Keycloak
    Keycloak-->>Gateway: return JWT access-,<br>refresh- and id-token
    deactivate Keycloak
    Gateway->>Redis: store Keycloak RenkuWebApplicationClient instance
    Gateway-->>User: redirect
    deactivate Gateway

    User->>Gateway: /api/auth/GitLab/login?redirect_url
    activate Gateway
    Gateway->>Redis: store GitLab RenkuWebApplicationClient instance
    Gateway-->>User: redirect
    deactivate Gateway
    
    User->>GitLab: /GitLab/oauth/authorize
    activate GitLab
    GitLab->>GitLab: redirect to Keycloak (KC)<br>if KC is set as identity provider<br>for GitLab
    GitLab-->>User: redirect
    deactivate GitLab

    User->>Gateway: /api/auth/GitLab/token
    activate Gateway
    Redis->>Gateway: get GitLab RenkuWebApplicationClient instance
    Gateway->>GitLab: /GitLab/oauth/token
    activate GitLab
    GitLab->>Gateway: return access- and refresh token
    deactivate GitLab
    Gateway->>Redis: store GitLab RenkuWebApplicationClient instance
    Gateway-->>User: redirect
    deactivate Gateway
    deactivate User
```
