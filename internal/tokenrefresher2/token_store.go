package tokenrefresher2

import (
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

// RefresherTokenStore is an interface used for refreshing tokens stored by the gateway
type RefresherTokenStore interface {
	models.AccessTokenGetter
	models.AccessTokenSetter
	models.RefreshTokenGetter
	models.RefreshTokenSetter
}
