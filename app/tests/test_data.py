TOKEN_PAYLOAD = {
    "jti": "ebb2b1cb-6176-483c-9671-88ced95f9a2f",
    "exp": 999999999999,
    "nbf": 0,
    "iat": 1528894957,
    "iss": "http://keycloak.renku.build:8080/auth/realms/Renku",
    "aud": "gateway",
    "sub": "5dbdeba7-e40f-42a7-b46b-6b8a07c65966",
    "typ": "Bearer",
    "azp": "gateway",
    "auth_time": 1528894957,
    "session_state": "899bfe3c-5a7e-4ea0-b340-b4179b297968",
    "acr": "1",
    "allowed-origins": [
        "http://gateway.renku.build/*",
        "http://localhost:5000/*"
    ],
    "realm_access": {
        "roles": [
            "uma_authorization"
        ]
    },
    "resource_access": {
        "account": {
            "roles": [
                "manage-account",
                "manage-account-links",
                "view-profile"
            ]
        }
    },
    "name": "Andreas Bleuler",
    "preferred_username": "ableuler",
    "given_name": "Andreas",
    "family_name": "Bleuler",
    "email": "andreas.bleuler@sdsc.ethz.ch"
}

PRIVATE_KEY = \
    '-----BEGIN RSA PRIVATE KEY-----\n' \
    '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\n' \
    '-----END RSA PRIVATE KEY-----'

PUBLIC_KEY = \
    '-----BEGIN PUBLIC KEY-----\n' \
    'MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAz9NEMFlsDjXOa6VuBjVz' \
    'vA3tRPJezepJt1CLNxZW1La7xbn1BxPQD1ccmxVNmlSnQd2xkS1oKsv8c7TcZY6m' \
    'Jma6UOKQqsjpI8ghDcCEk+l+xSh8AnuRIm7FQXY9nwdcxZsD+MLW6+HJLE4kAPob' \
    '8MjV05O6Nh85YF+8CGGW7Iv67gRHFzsaP3ofKcsyngs8vA+z5WpvMhv7NQtbw8ue' \
    'bdpH3sEmGaBYIdVZa5quiwzM1rQLQUw688dxuAFCmv0QpFQfO5lYeuRX9e7gRqDQ' \
    'kjG9nh0hnCKs7GVg8efafBuaUQlg038D2V1tVDWds4n+oICcmfYOUaspIWBa9QKe' \
    'Jiv/8qcdh3GbvNqFPWX/IVmZgrQGQ1c75VfEXY6wdXmQwwmMW+1bjJIRUtO6Z//Q' \
    'dggBrB5BjeS6XWafBVV3XtzkgSYiBmOmSLGlTXULyxhktpddGZO8Bg6HwLa2aVNp' \
    'neSOqcKkQObzeHKJm35tARGCRy6kbahy1kXnZ5MC1DIHlwHa9DK+TWC7mHC7vFxI' \
    'AwBEGFMTK5kP7Yg92x67JtjwR6w7T2zNwEfHGge9XwA28VcBUCZqBmeS3vsRw4h6' \
    'rmLQw3c+CD5jbviv9xMHw/XA77JEJd1XckKI8vwOI+ZQ2vcqYo2X6CUx5yHjirm2' \
    'bkr/hHS8VK7rmF9rCD2gQM0CAwEAAQ==\n' \
    '-----END PUBLIC KEY-----'
