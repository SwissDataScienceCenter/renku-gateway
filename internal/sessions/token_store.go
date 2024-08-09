package sessions

import "github.com/SwissDataScienceCenter/renku-gateway/internal/models"

type SessionTokenRepository interface {
	models.AccessTokenGetter
	models.AccessTokenSetter
	// AccessTokenRemover
	// RefreshTokenGetter
	models.RefreshTokenSetter
	// RefreshTokenRemover
	// IDTokenGetter
	models.IDTokenSetter
	// IDTokenRemover
}
