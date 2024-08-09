package tokenrefresher2

import (
	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
)

// RefresherTokenStore is an interface used for refreshing tokens stored by the gateway
type RefresherTokenStore interface {
	sessions.AccessTokenGetter
	sessions.AccessTokenSetter
	sessions.RefreshTokenGetter
	sessions.RefreshTokenSetter
}
