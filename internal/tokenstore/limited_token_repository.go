package tokenstore

import (
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

type LimitedTokenRepository interface {
	models.AccessTokenGetter
	models.AccessTokenSetter
	models.RefreshTokenGetter
	models.RefreshTokenSetter
}
