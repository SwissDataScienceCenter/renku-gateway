package models

import "context"

type TokenStoreInterface interface {
	FreshAccessTokenGetter
	AccessTokenSetter
	// AccessTokenRemover
	RefreshTokenGetter
	RefreshTokenSetter
	// RefreshTokenRemover
	FreshIDTokenGetter
	IDTokenSetter
	// IDTokenRemover
}

type FreshAccessTokenGetter interface {
	GetFreshAccessToken(ctx context.Context, tokenID string) (AuthToken, error)
}

type FreshIDTokenGetter interface {
	GetFreshIDToken(ctx context.Context, tokenID string) (AuthToken, error)
}
