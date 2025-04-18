package oidc

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

type ClientStore map[string]oidcClient

func (c ClientStore) AuthHandler(providerID string, state string) (http.HandlerFunc, error) {
	client, clientFound := c[providerID]
	if !clientFound {
		return nil, fmt.Errorf("cannot find the provider with ID %s", providerID)
	}
	return client.authHandler(state), nil
}

type CodeExchangeHandlerFunc func(callback TokenSetCallback) http.HandlerFunc

// Returns a http handler that will receive the authorization code from the identity provider.
// swap it for an access token and then pass the access and refresh token to the callback function.
func (c ClientStore) CodeExchangeHandler(providerID string) (CodeExchangeHandlerFunc, error) {
	client, clientFound := c[providerID]
	if !clientFound {
		return nil, fmt.Errorf("cannot find the provider with ID %s", providerID)
	}
	return func(callback TokenSetCallback) http.HandlerFunc {
		return client.codeExchangeHandler(callback)
	}, nil
}

func (c ClientStore) RefreshAccessToken(ctx context.Context, refreshToken models.AuthToken) (models.AuthTokenSet, error) {
	providerID := refreshToken.ProviderID
	client, clientFound := c[providerID]
	if !clientFound {
		return models.AuthTokenSet{}, fmt.Errorf("cannot find the provider with ID %s", providerID)
	}
	return client.refreshAccessToken(ctx, refreshToken)
}

func (c ClientStore) UserProfileURL(providerID string) (*url.URL, error) {
	client, clientFound := c[providerID]
	if !clientFound {
		return nil, fmt.Errorf("cannot find the provider with ID %s", providerID)
	}
	return client.userProfileURL()
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
