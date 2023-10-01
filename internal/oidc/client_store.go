package oidc

import (
	"fmt"
	"net/http"
	"os"

	"gopkg.in/yaml.v3"
)

type ClientStore map[string]Client

func (c ClientStore) CallbackHandler(providerID string, tokensHandler TokensHandler) (http.HandlerFunc, error) {
	client, clientFound := c[providerID]
	if !clientFound {
		return nil, fmt.Errorf("cannot find the provider with ID %s", providerID)
	}
	return func(rw http.ResponseWriter, r *http.Request) {
		client.CodeExchangeHandler(tokensHandler)(rw, r)
	}, nil
}

func (c ClientStore) AuthHandler(providerID string) (http.HandlerFunc, error) {
	client, clientFound := c[providerID]
	if !clientFound {
		return nil, fmt.Errorf("cannot find the provider with ID %s", providerID)
	}
	return client.AuthHandler(), nil
}

func NewClientStore(configs map[string]Config) (ClientStore, error) {
	var clients ClientStore
	for id, config := range configs {
		client, err := NewClient(config, id)
		if err != nil {
			return ClientStore{}, err
		}
		clients[id] = client
	}
	return clients, nil
}

func NewClientStoreFromFile(configFile string) (ClientStore, error) {
	f, err := os.Open(configFile)
	if err != nil {
		return ClientStore{}, err
	}
	var config map[string]Config
	err = yaml.NewDecoder(f).Decode(&config)
	if err != nil {
		return ClientStore{}, err
	}
	output, err := NewClientStore(config)
	if err != nil {
		return ClientStore{}, err
	}
	return output, nil
}
