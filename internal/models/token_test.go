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
	token := AuthToken{
		ID:         "123456",
		Value:      "secretValue",
		ExpiresAt:  time.Now().Add(time.Hour * 4),
		ProviderID: "providerId",
		SessionID:  "sessionId",
		Type:       AccessTokenType,
	}
	encToken, err := token.Encrypt(&encryptor)
	require.NoError(t, err)
	assert.Equal(t, token.Value+encryptSuffix, encToken.Value)
	encToken.Value = token.Value
	assert.Equal(t, token, encToken)
}

func TestDecrypt(t *testing.T) {
	encryptSuffix := "_encrypted"
	encryptor := MockEncryptor{encryptSuffix}
	token := AuthToken{
		ID:         "123456",
		Value:      "secretValue",
		ExpiresAt:  time.Now().Add(time.Hour * 4),
		ProviderID: "providerId",
		SessionID:  "sessionId",
		Type:       AccessTokenType,
	}
	encToken, err := token.Encrypt(&encryptor)
	require.NoError(t, err)
	decToken, err := encToken.Decrypt(&encryptor)
	require.NoError(t, err)
	assert.Equal(t, token.Value+encryptSuffix, encToken.Value)
	assert.Equal(t, token, decToken)
}

func TestNoEncryptor(t *testing.T) {
	token := AuthToken{
		ID:         "123456",
		Value:      "secretValue",
		ExpiresAt:  time.Now().Add(time.Hour * 4),
		ProviderID: "providerId",
		SessionID:  "sessionId",
		Type:       AccessTokenType,
	}
	encToken, err := token.Encrypt(nil)
	require.NoError(t, err)
	decToken, err := encToken.Decrypt(nil)
	require.NoError(t, err)
	assert.Equal(t, token, encToken)
	assert.Equal(t, token, decToken)
}

func TestTokenString(t *testing.T) {
	token := AuthToken{
		ID:         "123456",
		Value:      "secretValue",
		ProviderID: "providerId",
		SessionID:  "sessionId",
		Type:       AccessTokenType,
	}
	tokenString := token.String()
	assert.Contains(t, tokenString, "ID: 123456")
	assert.Contains(t, tokenString, "Value: redacted")
	assert.NotContains(t, tokenString, "secretValue")
	assert.Contains(t, tokenString, "ProviderID: providerId")
	assert.Contains(t, tokenString, "SessionID: sessionId")
	assert.Contains(t, tokenString, AccessTokenType)
}

func TestTokenNotExpired(t *testing.T) {
	token := AuthToken{
		ExpiresAt: time.Now().Add(time.Duration(5) * time.Minute),
	}
	assert.False(t, token.Expired())
}

func TestTokenExpired(t *testing.T) {
	token := AuthToken{
		ExpiresAt: time.Now().Add(-time.Duration(5) * time.Minute),
	}
	assert.True(t, token.Expired())
}

func TestTokenNoExpiry(t *testing.T) {
	token := AuthToken{
		ExpiresAt: time.Time{},
	}
	assert.True(t, token.ExpiresAt.IsZero())
	assert.False(t, token.Expired())
}

func TestTokenNotExpiresSoon(t *testing.T) {
	token := AuthToken{
		ExpiresAt: time.Now().Add(time.Duration(5) * time.Minute),
	}
	assert.False(t, token.ExpiresSoon(time.Duration(3)*time.Minute))
}

func TestTokenExpiresSoon(t *testing.T) {
	token := AuthToken{
		ExpiresAt: time.Now().Add(time.Duration(2) * time.Minute),
	}
	assert.True(t, token.ExpiresSoon(time.Duration(3)*time.Minute))
}

func TestTokenExpiresSoonNoExpiry(t *testing.T) {
	token := AuthToken{
		ExpiresAt: time.Time{},
	}
	assert.True(t, token.ExpiresAt.IsZero())
	assert.False(t, token.ExpiresSoon(time.Duration(3)*time.Minute))
}
