package main

import (
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

type TokenStore interface {
	models.AccessTokenGetter
	models.AccessTokenSetter
	models.AccessTokenRemover
	models.RefreshTokenGetter
	models.RefreshTokenSetter
	models.RefreshTokenRemover
}

type SessionStore interface {
	models.SessionGetter
	models.SessionSetter
	models.SessionRemover
}

type SessionTokenStore interface {
	SessionStore
	TokenStore
}
