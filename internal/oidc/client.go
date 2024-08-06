package oidc

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v2/pkg/http"
	"github.com/zitadel/oidc/v2/pkg/oidc"
	"golang.org/x/oauth2"
)

type oidcClient struct {
	client rp.RelyingParty
	id     string
}

func (c *oidcClient) getCodeExchangeCallback(tokensCallback models.TokensHandler) func(
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
		id, err := models.ULIDGenerator{}.ID()
		if err != nil {
			slog.Error("generating token ID failed in token exchange", "error", err, "requestID", r.Header.Get("X-Request-ID"))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		accessToken := models.AuthToken{
			ID:         id,
			Type:       models.AccessTokenType,
			Value:      tokens.AccessToken,
			TokenURL:   client.OAuthConfig().Endpoint.TokenURL,
			ExpiresAt:  tokens.Expiry,
			ProviderID: c.getID(),
		}
		refreshToken := models.AuthToken{
			ID:         id,
			Type:       models.RefreshTokenType,
			Value:      tokens.RefreshToken,
			TokenURL:   client.OAuthConfig().Endpoint.TokenURL,
			ProviderID: c.getID(),
		}
		idToken := models.AuthToken{
			ID:         id,
			Type:       models.IDTokenType,
			Value:      tokens.IDToken,
			ExpiresAt:  tokens.IDTokenClaims.GetExpiration(),
			ProviderID: c.getID(),
		}
		err = tokensCallback(accessToken, refreshToken, idToken)
		if err != nil {
			slog.Error("error when running tokens callback", "error", err, "requestID", r.Header.Get("X-Request-ID"))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// authHandler returns a http handler that can start the login flow and redirect
// to the identity provider /authorization page, setting all required parameters
// like state, client ID, secret, etc. We store the oAuth state values in the session
// in Redis so the function here just forwards the state that was provided from the session.
func (c *oidcClient) authHandler(state string) http.HandlerFunc {
	stateFunc := func() string {
		return state
	}
	return rp.AuthURLHandler(stateFunc, c.client)
}

// Returns a http handler that will receive the authorization code from the identity provider.
// swap it for an access token and then pass the access and refresh token to the callback function.
func (c *oidcClient) CodeExchangeHandler(tokensCallback models.TokensHandler) http.HandlerFunc {
	return rp.CodeExchangeHandler(c.getCodeExchangeCallback(tokensCallback), c.client)
}

func (c *oidcClient) getID() string {
	return c.id
}

func (c *oidcClient) startDeviceFlow(ctx context.Context) (*oauth2.DeviceAuthResponse, error) {
	// NOTE: the Zitadel OIDC library does not set this field when doing OIDC discovery automatically
	// And if this is not done here manually then the device flow all providers will not work
	c.client.OAuthConfig().Endpoint.DeviceAuthURL = c.client.GetDeviceAuthorizationEndpoint()
	return c.client.OAuthConfig().DeviceAuth(ctx)
}

// Verifies the signature only if the token is signed with RS256, checks the token is not expired, parses the claims and returns them
// NOTE: For Gitlab only the ID tokens can be parsed like this, access and refresh tokens are not JWTs
// NOTE: This will and should return a list of 3 tokens in the order in which they are defined in the function
func (c *oidcClient) verifyTokens(ctx context.Context, accessToken, refreshToken, idToken string) ([]models.AuthToken, error) {
	checkToken := func(val string, tokenID string, tokenType models.OauthTokenType, ks oidc.KeySet) (models.AuthToken, error) {
		claims := new(oidc.TokenClaims)
		payload, err := oidc.ParseToken(val, claims)
		if err != nil {
			return models.AuthToken{}, err
		}
		if claims.SignatureAlg == "RS256" {
			err = oidc.CheckSignature(ctx, val, payload, claims, []string{"RS256"}, ks)
			if err != nil {
				return models.AuthToken{}, err
			}
		}
		if tokenType != models.RefreshTokenType {
			err = oidc.CheckExpiration(claims, 0)
			if err != nil {
				return models.AuthToken{}, err
			}
		}
		output := models.AuthToken{ID: tokenID, Type: tokenType, Value: val, ExpiresAt: claims.GetExpiration(), TokenURL: c.client.OAuthConfig().Endpoint.TokenURL, ProviderID: c.getID()}
		return output, nil
	}

	ks := c.client.IDTokenVerifier().KeySet()
	tokenID, err := models.ULIDGenerator{}.ID()
	if err != nil {
		return []models.AuthToken{}, err
	}
	accessTokenParsed, err := checkToken(accessToken, tokenID, models.AccessTokenType, ks)
	if err != nil {
		slog.Info("OIDC", "error", err, "message", "cannot verify access token")
		return []models.AuthToken{}, err
	}
	refreshTokenParsed, err := checkToken(refreshToken, tokenID, models.RefreshTokenType, ks)
	if err != nil {
		slog.Info("OIDC", "error", err, "message", "cannot verify refresh token")
		return []models.AuthToken{}, err
	}
	if idToken == "" {
		return []models.AuthToken{accessTokenParsed, refreshTokenParsed, {}}, nil
	}
	idTokenParsed, err := checkToken(idToken, tokenID, models.IDTokenType, ks)
	if err != nil {
		slog.Info("OIDC", "error", err, "message", "cannot verify ID token")
		return []models.AuthToken{}, err
	}
	return []models.AuthToken{accessTokenParsed, refreshTokenParsed, idTokenParsed}, nil
}

// TODO
func (c *oidcClient) RefreshAccessToken(refreshToken models.AuthToken) error {
	_, err := rp.RefreshAccessToken(c.client, refreshToken.Value, "", "")
	if err != nil {
		return err
	}
	return nil
}

type clientOption func(*oidcClient) error

func withOIDCConfig(clientConfig config.OIDCClient) clientOption {
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
	return func(c *oidcClient) error {
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

func newClient(id string, options ...clientOption) (oidcClient, error) {
	client := oidcClient{id: id}
	for _, opt := range options {
		err := opt(&client)
		if err != nil {
			return oidcClient{}, err
		}
	}
	return client, nil
}
