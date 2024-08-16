package oidc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
)

type ClientStore map[string]oidcClient

func (c ClientStore) AuthHandler(providerID string, state string) (http.HandlerFunc, error) {
	client, clientFound := c[providerID]
	if !clientFound {
		return nil, fmt.Errorf("cannot find the provider with ID %s", providerID)
	}
	return client.authHandler(state), nil
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
