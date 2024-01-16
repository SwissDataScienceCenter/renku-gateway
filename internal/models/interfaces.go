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
	GetAccessToken(ctx context.Context, tokenID string) (OauthToken, error)
	GetAccessTokens(ctx context.Context, tokenIDs ...string) (map[string]OauthToken, error)
	GetExpiringAccessTokenIDs(ctx context.Context, expiryStart time.Time, expiryEnd time.Time) ([]string, error)
}

type AccessTokenSetter interface {
	SetAccessToken(context.Context, OauthToken) error
}

type AccessTokenRemover interface {
	RemoveAccessToken(context.Context, OauthToken) error
}

type RefreshTokenGetter interface {
	GetRefreshToken(ctx context.Context, tokenID string) (OauthToken, error)
	GetRefreshTokens(ctx context.Context, tokenIDs ...string) (map[string]OauthToken, error)
}

type RefreshTokenSetter interface {
	SetRefreshToken(context.Context, OauthToken) error
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

type TokensHandler func(accessToken, refreshToken OauthToken) error

type OIDCProviderStore interface {
	CallbackHandler(providerID string, tokensHandler TokensHandler) (http.HandlerFunc, error)
	AuthHandler(providerID string, state string) (http.HandlerFunc, error)
}

type OIDCProvider interface {
	AuthHandler(state string) http.HandlerFunc
	CodeExchangeHandler(tokensHandler TokensHandler) http.HandlerFunc
	ID() string
}
