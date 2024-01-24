package models

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionExpired(t *testing.T) {
	session, err := NewSession()
	require.NoError(t, err)
	assert.False(t, session.Expired())
	session.CreatedAt = time.Now().UTC().Add(-999 * time.Hour)
	assert.True(t, session.Expired())
}

func TestSetLoginURL(t *testing.T) {
	store := NewDummyDBAdapter()
	session, err := NewSession()
	require.NoError(t, err)
	session.sessionStore = &store
	assert.Equal(t, session.RedirectURL, "")
	url := "http://url1.com"
	err = session.SetRedirectURL(context.Background(), url)
	require.NoError(t, err)
	assert.Equal(t, url, session.RedirectURL)
}

func TestProviderIDs(t *testing.T) {
	session, err := NewSession(WithProviders("providerID1", "providerID2"))
	require.NoError(t, err)
	assert.Equal(t, 2, session.ProviderIDs.Len())
	assert.Equal(t, "providerID1", session.PeekProviderID())
	assert.Equal(t, "providerID2", session.ProviderIDs.Newest().Value)
	session, err = NewSession()
	require.NoError(t, err)
	store := NewDummyDBAdapter()
	session.sessionStore = &store
	assert.Equal(t, "", session.PeekProviderID())
	err = session.SetProviders(context.Background(), "providerID1", "providerID2")
	require.NoError(t, err)
	assert.Equal(t, 2, session.ProviderIDs.Len())
}

func TestAddTokenID(t *testing.T) {
	session, err := NewSession(WithProviders("provider1"))
	state := session.ProviderIDs.Oldest().Key
	require.NoError(t, err)
	store := NewDummyDBAdapter()
	session.tokenStore = &store
	session.sessionStore = &store
	assert.Len(t, session.TokenIDs, 0)
	at := OauthToken{ID: "1", Value: "access_token", ProviderID: "provider1", Type: AccessTokenType, ExpiresAt: time.Now().UTC().Add(time.Hour)}
	it := OauthToken{ID: "1", Value: "id_token", ProviderID: "provider1", Type: IDTokenType, ExpiresAt: time.Now().UTC().Add(time.Hour)}
	rt := OauthToken{ID: "1", Value: "refresh_token", ProviderID: "provider1", Type: RefreshTokenType, ExpiresAt: time.Now().UTC().Add(time.Hour)}
	err = session.SaveTokens(context.Background(), at, rt, it, state)
	require.NoError(t, err)
	assert.Len(t, session.TokenIDs, 1)
	rat, err := session.GetAccessToken(context.Background(), "provider1")
	require.NoError(t, err)
	assert.Equal(t, at, rat)
	rrt, err := session.tokenStore.GetRefreshToken(context.Background(), rt.ID)
	require.NoError(t, err)
	assert.Equal(t, rt, rrt)
	rit, err := session.tokenStore.GetIDToken(context.Background(), it.ID)
	require.NoError(t, err)
	assert.Equal(t, it, rit)
}
