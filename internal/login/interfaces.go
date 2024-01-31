package login

import (
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

type SessionTokenStore interface {
	models.SessionStore
	models.TokenStore
}
