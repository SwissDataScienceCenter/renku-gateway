package oidc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"golang.org/x/oauth2"
)

type ClientStore map[string]oidcClient

func (c ClientStore) CallbackHandler(providerID string, tokensHandler models.TokensHandler) (http.HandlerFunc, error) {
	client, clientFound := c[providerID]
	if !clientFound {
		return nil, fmt.Errorf("cannot find the provider with ID %s", providerID)
	}
	return func(rw http.ResponseWriter, r *http.Request) {
		client.CodeExchangeHandler(tokensHandler)(rw, r)
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

func (c ClientStore) StartDeviceFlow(ctx context.Context, providerID string) (*oauth2.DeviceAuthResponse, error) {
	client, clientFound := c[providerID]
	if !clientFound {
		return nil, fmt.Errorf("cannot find the provider with ID %s", providerID)
	}
	return client.startDeviceFlow(ctx)
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
