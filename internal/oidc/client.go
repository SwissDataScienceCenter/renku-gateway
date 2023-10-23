package oidc

import (
	"fmt"
	"net/http"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/idgenerators"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/google/uuid"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v2/pkg/http"
	"github.com/zitadel/oidc/v2/pkg/oidc"
)

type Client interface {
	AuthHandler() http.HandlerFunc
	CodeExchangeHandler(tokensHandler TokensHandler) http.HandlerFunc
	ID() string
}

type TokensHandler func(accessToken, refreshToken models.OauthToken) error

type Config struct {
	Issuer        string   `yaml:"issuer"`
	ClientID      string   `yaml:"clientId"`
	ClientSecret  string   `yaml:"clientSecret"`
	Scopes        []string `yaml:"scopes"`
	CookieHashKey string   `yaml:"cookieHashKey,omitempty"`
	CookieEncKey  string   `yaml:"cookieEncKey,omitempty"`
	CallbackURI   string   `yaml:"callbackURI"`
	NoPKCE        bool     `yaml:"noPKCE"`
	// UnsafeNoCookieHandler should NOT be set to true in production
	UnsafeNoCookieHandler bool `yaml:"unsafeNoCookieHandler"`
}

type zitadelClient struct {
	client rp.RelyingParty
	id     string
}

func (z *zitadelClient) getCodeExchangeCallback(tokensCallback TokensHandler) func(
	w http.ResponseWriter,
	r *http.Request,
	tokens *oidc.Tokens[*oidc.IDTokenClaims],
	state string,
	client rp.RelyingParty,
) {
	return func(
		w http.ResponseWriter,
		r *http.Request,
		tokens *oidc.Tokens[*oidc.IDTokenClaims],
		state string,
		client rp.RelyingParty,
	) {
		refreshTokenValue := tokens.RefreshToken
		accessTokenValue := tokens.AccessToken
		id, err := idgenerators.ULIDGenerator{}.ID()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		accessToken := models.OauthToken{
			ID:         id,
			Type:       models.AccessTokenType,
			Value:      accessTokenValue,
			TokenURL:   client.OAuthConfig().Endpoint.TokenURL,
			ExpiresAt:  tokens.Expiry,
			ProviderID: z.ID(),
		}
		var refreshTokenExpiresIn int
		if refreshTokenExpiresInRaw := tokens.Extra("refresh_expires_in"); refreshTokenExpiresInRaw != nil {
			refreshTokenExpiresIn = refreshTokenExpiresInRaw.(int)
		}
		var refreshTokenExpiry time.Time
		if refreshTokenExpiresIn > 0 {
			refreshTokenExpiry = time.Now().Add(time.Second * time.Duration(refreshTokenExpiresIn))
		}
		refreshToken := models.OauthToken{
			ID:         id,
			Type:       models.RefreshTokenType,
			Value:      refreshTokenValue,
			TokenURL:   client.OAuthConfig().Endpoint.TokenURL,
			ExpiresAt:  refreshTokenExpiry,
			ProviderID: z.ID(),
		}
		err = tokensCallback(accessToken, refreshToken)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

func (z *zitadelClient) AuthHandler() http.HandlerFunc {
	stateFunc := func() string {
		return uuid.NewString()
	}
	return rp.AuthURLHandler(stateFunc, z.client)
}

func (z *zitadelClient) CodeExchangeHandler(tokensCallback TokensHandler) http.HandlerFunc {
	return rp.CodeExchangeHandler(z.getCodeExchangeCallback(tokensCallback), z.client)
}

func (z *zitadelClient) ID() string {
	return z.id
}

func NewClient(config Config, id string) (Client, error) {
	options := []rp.Option{}
	if !config.NoPKCE || !config.UnsafeNoCookieHandler {
		cookieEncKey := []byte(config.CookieEncKey)
		cookieHashKey := []byte(config.CookieHashKey)
		if len(cookieEncKey) > 0 && !(len(cookieEncKey) == 16 || len(cookieEncKey) == 32) {
			return nil, fmt.Errorf(
				"Invalid length for oauth2 state cookie encryption key, got %d, but allowed sizes are 16 or 32",
				len(cookieEncKey),
			)
		}
		if len(cookieEncKey) == 0 {
			cookieEncKey = nil
		}
		if len(cookieHashKey) != 32 {
			return nil, fmt.Errorf(
				"Invalid length for oauth2 state cookie hash key, got %d, allowed size is 32",
				len(cookieHashKey),
			)
		}
		cookieHandler := httphelper.NewCookieHandler(cookieHashKey, cookieEncKey)
		options = append(options, rp.WithCookieHandler(cookieHandler))
		if !config.NoPKCE {
			options = append(options, rp.WithPKCE(cookieHandler))
		}
	}
	client, err := rp.NewRelyingPartyOIDC(
		config.Issuer,
		config.ClientID,
		config.ClientSecret,
		config.CallbackURI,
		config.Scopes,
		options...,
	)
	if err != nil {
		return nil, err
	}
	return &zitadelClient{client, id}, nil
}
