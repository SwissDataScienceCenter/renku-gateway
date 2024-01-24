package oidc

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v2/pkg/http"
	"github.com/zitadel/oidc/v2/pkg/oidc"
)

type Client struct {
	client rp.RelyingParty
	id     string
}

func (c *Client) getCodeExchangeCallback(tokensCallback models.TokensHandler) func(
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
		id, err := models.ULIDGenerator{}.ID()
		if err != nil {
			slog.Error("generating token ID failed in token exchange", "error", err, "requestID", r.Header.Get("X-Request-ID"))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		accessToken := models.OauthToken{
			ID:         id,
			Type:       models.AccessTokenType,
			Value:      accessTokenValue,
			TokenURL:   client.OAuthConfig().Endpoint.TokenURL,
			ExpiresAt:  tokens.Expiry,
			ProviderID: c.ID(),
		}
		var refreshTokenExpiresIn int
		var isInt bool
		if refreshTokenExpiresInRaw := tokens.Extra("refresh_expires_in"); refreshTokenExpiresInRaw != nil {
			refreshTokenExpiresIn, isInt = refreshTokenExpiresInRaw.(int)
			if !isInt {
				refreshTokenExpiresInStr, isStr := refreshTokenExpiresInRaw.(string)
				refreshTokenExpiresIn, err = strconv.Atoi(refreshTokenExpiresInStr)
				// refresh_expires_in is not a standard field so if we cannot parse it after a few tries
				// we just give up
				if isStr && err != nil {
					slog.Error("cannot parse expires_in of refresh token", "error", err, "requestID", r.Header.Get("X-Request-ID"))
					http.Error(w, "cannot parse expires_in of refresh token", http.StatusInternalServerError)
					return
				}
			}
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
			ProviderID: c.ID(),
		}
		idToken := models.OauthToken{
			ID: id,
			Type: models.IDTokenType,
			Value: tokens.IDToken,
			ExpiresAt: tokens.IDTokenClaims.GetExpiration(),
			ProviderID: c.ID(),
		}
		err = tokensCallback(accessToken, refreshToken, idToken)
		if err != nil {
			slog.Error("error when running tokens callback", "error", err, "requestID", r.Header.Get("X-Request-ID"))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// AuthHandler returns a http handler that can start the login flow and redirect
// to the identity provider /authorization page, setting all reaquired paramters
// like state, client ID, secret, etc. We store the oAuth state values in the session
// in Redis so the function here just forwards the state that was provided from the session.
func (c *Client) AuthHandler(state string) http.HandlerFunc {
	stateFunc := func() string {
		return state
	}
	return rp.AuthURLHandler(stateFunc, c.client)
}

// Returns a http handler that will receive the authorization code from the identity provider.
// swap it for an access token and then pass the access and refresh token to the callback function.
func (c *Client) CodeExchangeHandler(tokensCallback models.TokensHandler) http.HandlerFunc {
	return rp.CodeExchangeHandler(c.getCodeExchangeCallback(tokensCallback), c.client)
}

func (c *Client) ID() string {
	return c.id
}

type ClientOption func(*Client) error

func WithOIDCConfig(clientConfig config.OIDCClient) ClientOption {
	validateConfig := func(clientConfig config.OIDCClient) error {
		cookieEncKey := []byte(clientConfig.CookieEncodingKey)
		cookieHashKey := []byte(clientConfig.CookieHashKey)
		if len(cookieEncKey) > 0 && !(len(cookieEncKey) == 16 || len(cookieEncKey) == 32) {
			return fmt.Errorf(
				"Invalid length for oauth2 state cookie encryption key, got %d, but allowed sizes are 16 or 32",
				len(cookieEncKey),
			)
		}
		if len(cookieHashKey) > 0 && len(cookieHashKey) != 32 {
			return fmt.Errorf(
				"Invalid length for oauth2 state cookie hash key, got %d, allowed size is 32",
				len(cookieHashKey),
			)
		}
		return nil
	}
	makeClient := func(clientConfig config.OIDCClient) (rp.RelyingParty, error) {
		options := []rp.Option{}
		if !clientConfig.UnsafeNoCookieHandler {
			cookieEncKey := []byte(clientConfig.CookieEncodingKey)
			cookieHashKey := []byte(clientConfig.CookieHashKey)
			if len(cookieEncKey) == 0 {
				cookieEncKey = nil
			}
			cookieHandler := httphelper.NewCookieHandler(cookieHashKey, cookieEncKey)
			options = append(options, rp.WithCookieHandler(cookieHandler))
			if clientConfig.UsePKCE {
				options = append(options, rp.WithPKCE(cookieHandler))
			}
		}
		return rp.NewRelyingPartyOIDC(
			clientConfig.Issuer,
			clientConfig.ClientID,
			string(clientConfig.ClientSecret),
			clientConfig.CallbackURI,
			clientConfig.Scopes,
			options...,
		)
	}
	return func(c *Client) error {
		err := validateConfig(clientConfig)
		if err != nil {
			return err
		}
		client, err := makeClient(clientConfig)
		if err != nil {
			return err
		}
		c.client = client
		return nil
	}
}

func NewClient(id string, options ...ClientOption) (Client, error) {
	client := Client{id: id}
	for _, opt := range options {
		err := opt(&client)
		if err != nil {
			return Client{}, err
		}
	}
	return client, nil
}
