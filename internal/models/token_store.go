package models

import "context"

type TokenStoreInterface interface {
	FreshAccessTokenGetter
	AccessTokenSetter
	// AccessTokenRemover
	// RefreshTokenGetter
	RefreshTokenSetter
	// RefreshTokenRemover
	// IDTokenGetter
	IDTokenSetter
	// IDTokenRemover
}

type FreshAccessTokenGetter interface {
	GetFreshAccessToken(ctx context.Context, tokenID string) (AuthToken, error)
}
