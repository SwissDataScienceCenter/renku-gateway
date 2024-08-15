package oidc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
	"github.com/zitadel/oidc/v2/pkg/oidc"
	"golang.org/x/oauth2"
)

type ClientStore map[string]oidcClient

func (c ClientStore) CallbackHandler(providerID string, callback TokenSetCallback) (http.HandlerFunc, error) {
	client, clientFound := c[providerID]
	if !clientFound {
		return nil, fmt.Errorf("cannot find the provider with ID %s", providerID)
	}
	return func(rw http.ResponseWriter, r *http.Request) {
		client.CodeExchangeHandler(callback)(rw, r)
	}, nil
}

func (c ClientStore) AuthHandler(providerID string, state string) (http.HandlerFunc, error) {
	client, clientFound := c[providerID]
	if !clientFound {
		return nil, fmt.Errorf("cannot find the provider with ID %s", providerID)
	}
	return client.authHandler(state), nil
}

func (c ClientStore) VerifyTokens(ctx context.Context, providerID, accessToken, refreshToken, idToken string) ([]models.AuthToken, error) {
	client, clientFound := c[providerID]
	if !clientFound {
		return []models.AuthToken{}, fmt.Errorf("cannot find the provider with ID %s", providerID)
	}
	return client.verifyTokens(ctx, accessToken, refreshToken, idToken)
}

func (c ClientStore) VerifyAccessToken(ctx context.Context, providerID, accessToken string) (oidc.TokenClaims, error) {
	client, clientFound := c[providerID]
	if !clientFound {
		return oidc.TokenClaims{}, fmt.Errorf("cannot find the provider with ID %s", providerID)
	}
	return client.verifyAccessToken(ctx, accessToken)
}

func (c ClientStore) StartDeviceFlow(ctx context.Context, providerID string) (*oauth2.DeviceAuthResponse, error) {
	client, clientFound := c[providerID]
	if !clientFound {
		return nil, fmt.Errorf("cannot find the provider with ID %s", providerID)
	}
	return client.startDeviceFlow(ctx)
}

func (c ClientStore) RefreshAccessToken(ctx context.Context, refreshToken models.AuthToken) (sessions.AuthTokenSet, error) {
	providerID := refreshToken.ProviderID
	client, clientFound := c[providerID]
	if !clientFound {
		return sessions.AuthTokenSet{}, fmt.Errorf("cannot find the provider with ID %s", providerID)
	}
	return client.refreshAccessToken(ctx, refreshToken)
}

func NewClientStore(configs map[string]config.OIDCClient) (ClientStore, error) {
	var clients = ClientStore{}
	for id, config := range configs {
		client, err := newClient(id, withOIDCConfig(config))
		if err != nil {
			return ClientStore{}, err
		}
		clients[id] = client
	}
	return clients, nil
}
