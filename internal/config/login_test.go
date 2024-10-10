package config

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getValidLoginConfig(t *testing.T) LoginConfig {
	renkuBaseUrl, err := url.Parse("https://renku.example.org")
	require.NoError(t, err)
	return LoginConfig{
		RenkuBaseURL:        renkuBaseUrl,
		LoginRoutesBasePath: "/api/auth",
		TokenEncryption: TokenEncryptionConfig{
			Enabled:   true,
			SecretKey: "eBfR0WfHBTrRrVdLpsTYmWtPwJfQqOEq",
		},
	}
}

func TestValidLoginConfig(t *testing.T) {
	config := getValidLoginConfig(t)

	err := config.Validate(Production)

	assert.NoError(t, err)
}

func TestInvalidTokenEncryptionSecretKey(t *testing.T) {
	config := getValidLoginConfig(t)
	config.TokenEncryption.SecretKey = "invalid-key"

	err := config.Validate(Production)

	assert.ErrorContains(t, err, "token encryption key has to be 32 bytes long, the provided one is 11 long")
}

func TestInvalidProviderName(t *testing.T) {
	config := getValidLoginConfig(t)
	config.Providers = map[string]OIDCClient{
		"invalid-name": OIDCClient{},
	}

	err := config.Validate(Production)

	assert.ErrorContains(t, err, "unknown provider id invalid-name (must be one of renku or gitlab)")
}

func TestInvalidProviderUnsafeNoCookieHandler(t *testing.T) {
	config := getValidLoginConfig(t)
	config.Providers = map[string]OIDCClient{
		"renku": OIDCClient{
			UnsafeNoCookieHandler: true,
		},
	}

	err := config.Validate(Production)

	assert.ErrorContains(t, err, "provider renku cannot be configured without a cookie handler in production")
}
