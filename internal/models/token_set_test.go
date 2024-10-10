package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateTokensType(t *testing.T) {
	tokenID := "my-token"
	accessToken := AuthToken{
		ID:   tokenID,
		Type: AccessTokenType,
	}
	refreshToken := AuthToken{
		ID:   tokenID,
		Type: RefreshTokenType,
	}
	idToken := AuthToken{
		ID:   tokenID,
		Type: IDTokenType,
	}
	tokenSet := AuthTokenSet{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IDToken:      idToken,
	}

	err := tokenSet.ValidateTokensType()

	assert.NoError(t, err)
}

func TestInvalidID(t *testing.T) {
	tokenID := "my-token"
	accessToken := AuthToken{
		ID:   tokenID,
		Type: AccessTokenType,
	}
	refreshToken := AuthToken{
		ID:   "invalid-id",
		Type: RefreshTokenType,
	}
	idToken := AuthToken{
		ID:   tokenID,
		Type: IDTokenType,
	}
	tokenSet := AuthTokenSet{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IDToken:      idToken,
	}

	err := tokenSet.ValidateTokensType()

	assert.ErrorContains(t, err, "tokens in a set should have the same ID")
}

func TestInvalidProviderID(t *testing.T) {
	tokenID := "my-token"
	providerID := "provider"
	accessToken := AuthToken{
		ID:         tokenID,
		ProviderID: providerID,
		Type:       AccessTokenType,
	}
	refreshToken := AuthToken{
		ID:         tokenID,
		ProviderID: "invalid-provider",
		Type:       RefreshTokenType,
	}
	idToken := AuthToken{
		ID:         tokenID,
		ProviderID: providerID,
		Type:       IDTokenType,
	}
	tokenSet := AuthTokenSet{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IDToken:      idToken,
	}

	err := tokenSet.ValidateTokensType()

	assert.ErrorContains(t, err, "tokens in a set should have the same provider ID")
}

func TestInvalidAccessToken(t *testing.T) {
	tokenID := "my-token"
	accessToken := AuthToken{
		ID:   tokenID,
		Type: RefreshTokenType,
	}
	refreshToken := AuthToken{
		ID:   tokenID,
		Type: RefreshTokenType,
	}
	idToken := AuthToken{
		ID:   tokenID,
		Type: IDTokenType,
	}
	tokenSet := AuthTokenSet{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IDToken:      idToken,
	}

	err := tokenSet.ValidateTokensType()

	assert.ErrorContains(t, err, "invalid type RefreshToken for access token my-token")
}

func TestInvalidRefreshToken(t *testing.T) {
	tokenID := "my-token"
	accessToken := AuthToken{
		ID:   tokenID,
		Type: AccessTokenType,
	}
	refreshToken := AuthToken{
		ID:   tokenID,
		Type: AccessTokenType,
	}
	idToken := AuthToken{
		ID:   tokenID,
		Type: IDTokenType,
	}
	tokenSet := AuthTokenSet{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IDToken:      idToken,
	}

	err := tokenSet.ValidateTokensType()

	assert.ErrorContains(t, err, "invalid type AccessToken for refresh token my-token")
}

func TestInvalidIDToken(t *testing.T) {
	tokenID := "my-token"
	accessToken := AuthToken{
		ID:   tokenID,
		Type: AccessTokenType,
	}
	refreshToken := AuthToken{
		ID:   tokenID,
		Type: RefreshTokenType,
	}
	idToken := AuthToken{
		ID:   tokenID,
		Type: AccessTokenType,
	}
	tokenSet := AuthTokenSet{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IDToken:      idToken,
	}

	err := tokenSet.ValidateTokensType()

	assert.ErrorContains(t, err, "invalid type AccessToken for ID token my-token")
}
