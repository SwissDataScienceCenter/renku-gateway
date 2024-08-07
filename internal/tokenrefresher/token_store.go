package tokenrefresher

import (
	"context"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
)

// RefresherTokenStore is an interface used for refreshing tokens stored by the gateway
type RefresherTokenStore interface {
	// GetRefreshToken(context.Context, string) (models.AuthToken, error)
	// GetAccessToken(context.Context, string) (models.AuthToken, error)
	// SetRefreshToken(context.Context, models.AuthToken) error
	// SetAccessToken(context.Context, models.AuthToken) error
	// GetExpiringAccessTokenIDs(context.Context, time.Time, time.Time) ([]string, error)
	sessions.AccessTokenGetter
	sessions.AccessTokenSetter
	sessions.RefreshTokenGetter
	sessions.RefreshTokenSetter
	ExpiringAccessTokensGetter
}

type ExpiringAccessTokensGetter interface {
	GetExpiringAccessTokenIDs(ctx context.Context, expiryEnd time.Time) ([]string, error)
}
