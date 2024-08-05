package sessions

import (
	"context"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

type TokenStore2 interface {
	AccessTokenGetter
	AccessTokenSetter
	// AccessTokenRemover
	// RefreshTokenGetter
	RefreshTokenSetter
	// RefreshTokenRemover
	// IDTokenGetter
	IDTokenSetter
	// IDTokenRemover
}

type AccessTokenGetter interface {
	GetAccessToken(ctx context.Context, tokenID string) (models.AuthToken, error)
}

type AccessTokenSetter interface {
	SetAccessToken(ctx context.Context, token models.AuthToken) error
}

type AccessTokenRemover interface {
	RemoveAccessToken(ctx context.Context, tokenID string) error
}

type RefreshTokenGetter interface {
	GetRefreshToken(ctx context.Context, tokenID string) (models.AuthToken, error)
}

type RefreshTokenSetter interface {
	SetRefreshToken(ctx context.Context, token models.AuthToken) error
}

type RefreshTokenRemover interface {
	RemoveRefreshToken(ctx context.Context, tokenID string) error
}

type IDTokenGetter interface {
	GetIDToken(ctx context.Context, tokenID string) (models.AuthToken, error)
}

type IDTokenSetter interface {
	SetIDToken(ctx context.Context, token models.AuthToken) error
}

type IDTokenRemover interface {
	RemoveIDToken(ctx context.Context, tokenID string) error
}
