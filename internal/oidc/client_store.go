package oidc

import (
	"fmt"
	"net/http"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

type ClientStore map[string]Client

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
	return client.AuthHandler(state), nil
}

func NewClientStore(configs map[string]config.OIDCClient) (ClientStore, error) {
	var clients = ClientStore{}
	for id, config := range configs {
		client, err := NewClient(id, WithOIDCConfig(config))
		if err != nil {
			return ClientStore{}, err
		}
		clients[id] = client
	}
	return clients, nil
}
