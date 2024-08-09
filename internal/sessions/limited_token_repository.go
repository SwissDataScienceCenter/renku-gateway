package sessions

import (
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

type LimitedTokenRepository interface {
	models.AccessTokenSetter
	// AccessTokenRemover
	// RefreshTokenGetter
	models.RefreshTokenSetter
	// RefreshTokenRemover
	// IDTokenGetter
	models.IDTokenSetter
	// IDTokenRemover
}
