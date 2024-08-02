package models

import (
	"context"
	"net/http"
	"time"
)

type Encryptor interface {
	Encrypt(value string) (encrypted string, err error)
	Decrypt(value string) (decrypted string, err error)
}

type IDGenerator interface {
	ID() (string, error)
}

type AccessTokenGetter interface {
	GetAccessToken(ctx context.Context, tokenID string) (AuthToken, error)
	GetAccessTokens(ctx context.Context, tokenIDs ...string) (map[string]AuthToken, error)
	GetExpiringAccessTokenIDs(ctx context.Context, expiryStart time.Time, expiryEnd time.Time) ([]string, error)
}

type AccessTokenSetter interface {
	SetAccessToken(context.Context, AuthToken) error
}

type AccessTokenRemover interface {
	RemoveAccessToken(context.Context, AuthToken) error
}

type IDTokenGetter interface {
	GetIDToken(ctx context.Context, tokenID string) (AuthToken, error)
	GetIDTokens(ctx context.Context, tokenIDs ...string) (map[string]AuthToken, error)
}

type IDTokenSetter interface {
	SetIDToken(context.Context, AuthToken) error
}

type IDTokenRemover interface {
	RemoveIDToken(context.Context, AuthToken) error
}

type RefreshTokenGetter interface {
	GetRefreshToken(ctx context.Context, tokenID string) (AuthToken, error)
	GetRefreshTokens(ctx context.Context, tokenIDs ...string) (map[string]AuthToken, error)
}

type RefreshTokenSetter interface {
	SetRefreshToken(context.Context, AuthToken) error
}

type RefreshTokenRemover interface {
	RemoveRefreshToken(context.Context, string) error
}

type SessionGetter interface {
	GetSession(context.Context, string) (Session, error)
}

type SessionSetter interface {
	SetSession(context.Context, Session) error
}

type SessionRemover interface {
	RemoveSession(context.Context, string) error
}

type TokensHandler func(accessToken, refreshToken, idToken AuthToken) error

type OIDCProviderStore interface {
	CallbackHandler(providerID string, tokensHandler TokensHandler) (http.HandlerFunc, error)
	AuthHandler(providerID string, state string) (http.HandlerFunc, error)
}

type OIDCProvider interface {
	AuthHandler(state string) http.HandlerFunc
	CodeExchangeHandler(tokensHandler TokensHandler) http.HandlerFunc
	ID() string
}
