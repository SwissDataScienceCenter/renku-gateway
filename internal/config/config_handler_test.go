package config

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createSecretFile(fpath string) error {
	contents := `---
login:
  tokenEncryption:
    secretKey: secret-key-from-secret-file
  providers:
    id1:
      clientSecret: client-secret-from-secret-file
      cookieEncodingKey: enc-key-from-secret-file
      cookieHashKey: hash-key-from-secret-file
`
	err := os.WriteFile(fpath, []byte(contents), 0666)
	return err
}

func TestReadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CONFIG_LOCATION", tmpDir)
	err := createSecretFile(path.Join(tmpDir, "secret_config.yaml"))
	require.NoError(t, err)
	ch := NewConfigHandler()
	config, err := ch.Config()
	require.NoError(t, err)
	assert.NotEqual(t, config, Config{})
	assert.Len(t, config.Login.Providers, 1)
	assert.Equal(t, "https://renkulab.io", config.Revproxy.RenkuBaseURL.String())
	assert.Equal(t, "secret-key-from-secret-file", config.Login.TokenEncryption.SecretKey)
	assert.Equal(t, "client-secret-from-secret-file", config.Login.Providers["id1"].ClientSecret)
	assert.Equal(t, "enc-key-from-secret-file", config.Login.Providers["id1"].CookieEncodingKey)
	assert.Equal(t, "hash-key-from-secret-file", config.Login.Providers["id1"].CookieHashKey)
	assert.Equal(t, true, config.Login.Providers["id1"].Default)
}

func TestReadConfigWithEnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CONFIG_LOCATION", tmpDir)
	err := createSecretFile(path.Join(tmpDir, "secret_config.yaml"))
	require.NoError(t, err)
	t.Setenv("LOGIN_PROVIDERS_ID1_CLIENTSECRET", "env-var-secret")
	t.Setenv("REVPROXY_RENKUBASEURL", "https://dev.renku.ch")
	ch := NewConfigHandler()
	config, err := ch.Config()
	require.NoError(t, err)
	assert.NotEqual(t, config, Config{})
	assert.Len(t, config.Login.Providers, 1)
	assert.Equal(t, "https://dev.renku.ch", config.Revproxy.RenkuBaseURL.String())
	assert.Equal(t, "secret-key-from-secret-file", config.Login.TokenEncryption.SecretKey)
	assert.Equal(t, "env-var-secret", config.Login.Providers["id1"].ClientSecret)
	assert.Equal(t, "enc-key-from-secret-file", config.Login.Providers["id1"].CookieEncodingKey)
	assert.Equal(t, "hash-key-from-secret-file", config.Login.Providers["id1"].CookieHashKey)
	assert.Equal(t, true, config.Login.Providers["id1"].Default)
}
