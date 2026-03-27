package models

import (
	"context"
	"time"
)

// TokenRefreshRepository represents the persistence interface used to keep refresh tokens alive
type TokenRefreshRepository interface {
	TokenRefreshExpiryGetter
}

type TokenRefreshExpiryGetter interface {
	// GetExpiringRefreshTokenIDs returns the token IDs of refresh tokens which will expire in the time range [from, to]
	GetExpiringRefreshTokenIDs(ctx context.Context, from, to time.Time) ([]string, error)
}
