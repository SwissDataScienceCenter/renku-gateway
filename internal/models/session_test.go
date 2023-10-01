package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionExpired(t *testing.T) {
	session, err := NewSession(time.Hour, []string{"providerID1"})
	require.NoError(t, err)
	assert.False(t, session.Expired())
	session.ExpiresAt = time.Now().Add(-8 * time.Hour)
	assert.True(t, session.Expired())
}

func TestSetLoginURL(t *testing.T) {
	session, err := NewSession(time.Hour, []string{"providerID1"})
	require.NoError(t, err)
	assert.Equal(t, session.RedirectURL, "")
	url := "http://url1"
	session.SetRedirectURL(url)
	assert.Equal(t, url, session.RedirectURL)
}

func TestProviderIDs(t *testing.T) {
	session, err := NewSession(time.Hour, []string{"providerID1"})
	require.NoError(t, err)
	assert.Len(t, session.LoginWithProviders, 1)
	assert.Equal(t, "providerID1", session.PopProviderID())
	assert.Len(t, session.LoginWithProviders, 0)
	assert.Equal(t, "", session.PopProviderID())
	providerIDs := SerializableStringSlice{"providerID2", "providerID3"}
	session.SetProviderIDs(providerIDs)
	assert.Equal(t, providerIDs, session.LoginWithProviders)
	assert.Equal(t, providerIDs[0], session.PopProviderID())
	assert.Equal(t, providerIDs[1:], session.LoginWithProviders)
}

func TestCodeVerifiers(t *testing.T) {
	session, err := NewSession(time.Hour, []string{"providerID1"})
	require.NoError(t, err)
	assert.Len(t, session.CodeVerifiers, 0)
	assert.Equal(t, "", session.PopCodeVerifier())
	codeVerifiers := SerializableStringSlice{"code1", "code2"}
	session.SetCodeVerifiers(codeVerifiers)
	assert.Equal(t, codeVerifiers, session.CodeVerifiers)
	assert.Equal(t, codeVerifiers[0], session.PopCodeVerifier())
	assert.Equal(t, codeVerifiers[1:], session.CodeVerifiers)
	assert.Equal(t, codeVerifiers[1], session.PopCodeVerifier())
	assert.Equal(t, SerializableStringSlice{}, session.CodeVerifiers)
}

func TestAddTokenID(t *testing.T) {
	session, err := NewSession(time.Hour, []string{"providerID1"})
	require.NoError(t, err)
	assert.Len(t, session.TokenIDs, 0)
	session.AddTokenID("test1")
	assert.Equal(t, SerializableStringSlice{"test1"}, session.TokenIDs)
}
