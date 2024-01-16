package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadConfig(t *testing.T) {
	ch := NewConfigHandler()
	config, err := ch.Config()
	require.NoError(t, err)
	assert.NotEqual(t, config, Config{})
	assert.Len(t, config.Login.Providers, 1)
	assert.Equal(t, "secret-key-from-secret-file", config.Login.TokenEncryption.SecretKey)
	assert.Equal(t, "client-secret-from-secret-file", config.Login.Providers["id1"].ClientSecret)
	assert.Equal(t, "enc-key-from-secret-file", config.Login.Providers["id1"].CookieEncodingKey)
	assert.Equal(t, "hash-key-from-secret-file", config.Login.Providers["id1"].CookieHashKey)
	assert.Equal(t, true, config.Login.Providers["id1"].Default)
}
