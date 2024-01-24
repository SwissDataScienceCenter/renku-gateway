package oidc

import (
	"testing"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestClientStore(t *testing.T) {
	client1 := Client{
		client: newMockRelyingParty("https://token.url"),
		id:     "id1",
	}
	client2 := Client{
		client: newMockRelyingParty("https://token.url"),
		id:     "id2",
	}
	clientStore := ClientStore{client1.id: client1, client2.id: client2}
	_, err := clientStore.CallbackHandler("id1", func(accessToken, refreshToken, idToken models.OauthToken) error { return nil })
	assert.NoError(t, err)
	_, err = clientStore.CallbackHandler("id2", func(accessToken, refreshToken, idToken models.OauthToken) error { return nil })
	assert.NoError(t, err)
	_, err = clientStore.CallbackHandler(
		"missing",
		func(accessToken, refreshToken, idToken models.OauthToken) error { return nil },
	)
	assert.Error(t, err)
}
