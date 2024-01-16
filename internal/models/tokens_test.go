package models

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockEncryptor struct {
	suffix string
}

func (m *MockEncryptor) Encrypt(value string) (encrypted string, err error) {
	return value + m.suffix, nil
}

func (m *MockEncryptor) Decrypt(value string) (decrypted string, err error) {
	return strings.TrimSuffix(value, m.suffix), nil
}

func TestEncrypt(t *testing.T) {
	encryptSuffix := "_encrypted"
	encryptor := MockEncryptor{encryptSuffix}
	token := OauthToken{
		ID:         "123456",
		Value:      "secretValue",
		ExpiresAt:  time.Now().Add(time.Hour * 4),
		ProviderID: "providerId",
		Type:       AccessTokenType,
		encryptor:  &encryptor,
	}
	encToken, err := token.Encrypt()
	require.NoError(t, err)
	assert.Equal(t, token.Value+encryptSuffix, encToken.Value)
	encToken.Value = token.Value
	assert.Equal(t, token, encToken)
}

func TestDecrypt(t *testing.T) {
	encryptSuffix := "_encrypted"
	encryptor := MockEncryptor{encryptSuffix}
	token := OauthToken{
		ID:         "123456",
		Value:      "secretValue",
		ExpiresAt:  time.Now().Add(time.Hour * 4),
		ProviderID: "providerId",
		Type:       AccessTokenType,
		encryptor:  &encryptor,
	}
	encToken, err := token.Encrypt()
	require.NoError(t, err)
	decToken, err := encToken.Decrypt()
	require.NoError(t, err)
	assert.Equal(t, token.Value+encryptSuffix, encToken.Value)
	assert.Equal(t, token, decToken)
}

func TestNoEncryptor(t *testing.T) {
	token := OauthToken{
		ID:         "123456",
		Value:      "secretValue",
		ExpiresAt:  time.Now().Add(time.Hour * 4),
		ProviderID: "providerId",
		Type:       AccessTokenType,
	}
	encToken, err := token.Encrypt()
	require.NoError(t, err)
	decToken, err := encToken.Decrypt()
	require.NoError(t, err)
	assert.Equal(t, token, encToken)
	assert.Equal(t, token, decToken)
}

func TestTokenEquality(t *testing.T) {
	expiresAt := time.Now().UTC().Add(time.Hour * 4)
	enc := MockEncryptor{"_enc"}
	token1 := OauthToken{
		ID:         "123456",
		Value:      "secretValue",
		ExpiresAt:  expiresAt,
		TokenURL:   "http://some.url.com/token",
		ProviderID: "providerId",
		Type:       AccessTokenType,
	}
	token2 := OauthToken{
		ID:         "123456",
		Value:      "secretValue",
		ExpiresAt:  expiresAt,
		TokenURL:   "http://some.url.com/token",
		ProviderID: "providerId",
		Type:       AccessTokenType,
		encryptor:  &enc,
	}
	// If the encryptor is not the same then the tokens are not equal
	assert.False(t, token1 == token2)
	token3 := OauthToken{
		ID:         "123456",
		Value:      "secretValue",
		ExpiresAt:  expiresAt,
		TokenURL:   "http://some.url.com/token",
		ProviderID: "providerId",
		Type:       AccessTokenType,
		encryptor:  &enc,
	}
	assert.True(t, token2 == token3)
	enc2 := MockEncryptor{"_enc"}
	token4 := OauthToken{
		ID:         "123456",
		Value:      "secretValue",
		ExpiresAt:  expiresAt,
		TokenURL:   "http://some.url.com/token",
		ProviderID: "providerId",
		Type:       AccessTokenType,
		encryptor:  &enc2,
	}
	assert.False(t, token3 == token4)
}
