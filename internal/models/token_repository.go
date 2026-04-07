package models

import (
	"context"
)

// TokenRepository represents the interface used to persist tokens
type TokenRepository interface {
	AccessTokenGetter
	AccessTokenSetter
	// AccessTokenRemover
	RefreshTokenGetter
	RefreshTokenSetter
	// RefreshTokenRemover
	IDTokenGetter
	IDTokenSetter
	// IDTokenRemover
}

type AccessTokenGetter interface {
	GetAccessToken(ctx context.Context, tokenID string) (AuthToken, error)
}

type AccessTokenSetter interface {
	SetAccessToken(ctx context.Context, token AuthToken) error
}

type AccessTokenRemover interface {
	RemoveAccessToken(ctx context.Context, tokenID string) error
}

type RefreshTokenGetter interface {
	GetRefreshToken(ctx context.Context, tokenID string) (AuthToken, error)
}

type RefreshTokenSetter interface {
	SetRefreshToken(ctx context.Context, token AuthToken) error
}

type RefreshTokenRemover interface {
	RemoveRefreshToken(ctx context.Context, tokenID string) error
}

type IDTokenGetter interface {
	GetIDToken(ctx context.Context, tokenID string) (AuthToken, error)
}

type IDTokenSetter interface {
	SetIDToken(ctx context.Context, token AuthToken) error
}

type IDTokenRemover interface {
	RemoveIDToken(ctx context.Context, tokenID string) error
}
