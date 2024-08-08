package oidc

import (
	"testing"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
	"github.com/stretchr/testify/assert"
)

func TestClientStore(t *testing.T) {
	client1 := oidcClient{
		client: newMockRelyingParty("https://token.url"),
		id:     "id1",
	}
	client2 := oidcClient{
		client: newMockRelyingParty("https://token.url"),
		id:     "id2",
	}
	clientStore := ClientStore{client1.id: client1, client2.id: client2}
	_, err := clientStore.CallbackHandler("id1", func(tokenSet sessions.AuthTokenSet) error { return nil })
	assert.NoError(t, err)
	_, err = clientStore.CallbackHandler("id2", func(tokenSet sessions.AuthTokenSet) error { return nil })
	assert.NoError(t, err)
	_, err = clientStore.CallbackHandler(
		"missing",
		func(tokenSet sessions.AuthTokenSet) error { return nil },
	)
	assert.Error(t, err)
}
