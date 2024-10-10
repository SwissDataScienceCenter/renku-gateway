package oidc

import (
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

type TokenSetCallback func(tokenSet models.AuthTokenSet) error
