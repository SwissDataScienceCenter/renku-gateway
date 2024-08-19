package oidc

import (
	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
)

type TokenSetCallback func(tokenSet sessions.AuthTokenSet) error
