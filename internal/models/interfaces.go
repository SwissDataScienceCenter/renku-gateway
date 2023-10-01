package models

import (
	"context"
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
	GetAccessToken(context.Context, string) (OauthToken, error)
	GetAccessTokens(context.Context, ...string) (map[string]OauthToken, error)
	GetExpiringAccessTokenIDs(context.Context, time.Time, time.Time) ([]string, error)
}

type AccessTokenSetter interface {
	SetAccessToken(context.Context, OauthToken) error
}

type AccessTokenRemover interface {
	RemoveAccessToken(context.Context, OauthToken) error
}

type RefreshTokenGetter interface {
	GetRefreshToken(context.Context, string) (OauthToken, error)
	GetRefreshTokens(context.Context, ...string) (map[string]OauthToken, error)
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
