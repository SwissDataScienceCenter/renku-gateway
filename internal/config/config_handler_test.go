package config

import (
	"log/slog"
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
    renku:
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
	providerID := "renku"
	ch := NewConfigHandler()
	config, err := ch.Config()
	require.NoError(t, err)
	assert.NotEqual(t, config, Config{})
	assert.Len(t, config.Login.Providers, 1)
	assert.Equal(t, "https://renkulab.io", config.Revproxy.RenkuBaseURL.String())
	assert.Equal(t, RedactedString("secret-key-from-secret-file"), config.Login.TokenEncryption.SecretKey)
	assert.Equal(t, RedactedString("client-secret-from-secret-file"), config.Login.Providers[providerID].ClientSecret)
	assert.Equal(t, RedactedString("enc-key-from-secret-file"), config.Login.Providers[providerID].CookieEncodingKey)
	assert.Equal(t, RedactedString("hash-key-from-secret-file"), config.Login.Providers[providerID].CookieHashKey)
	assert.Equal(t, true, config.Login.Providers[providerID].Default)
}

func TestReadConfigWithEnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CONFIG_LOCATION", tmpDir)
	err := createSecretFile(path.Join(tmpDir, "secret_config.yaml"))
	require.NoError(t, err)
	providerID := "renku"
	t.Setenv("GATEWAY_LOGIN_PROVIDERS_RENKU_CLIENTSECRET", "env-var-secret")
	t.Setenv("GATEWAY_REVPROXY_RENKUBASEURL", "https://dev.renku.ch")
	t.Setenv("GATEWAY_LOGIN_TOKENENCRYPTION_SECRETKEY", "token-encryption-key-12345678910")
	ch := NewConfigHandler()
	config, err := ch.Config()
	require.NoError(t, err)
	assert.NotEqual(t, config, Config{})
	assert.Len(t, config.Login.Providers, 1)
	assert.Equal(t, "https://dev.renku.ch", config.Revproxy.RenkuBaseURL.String())
	assert.Equal(t, RedactedString("env-var-secret"), config.Login.Providers[providerID].ClientSecret)
	assert.Equal(t, RedactedString("enc-key-from-secret-file"), config.Login.Providers[providerID].CookieEncodingKey)
	assert.Equal(t, RedactedString("hash-key-from-secret-file"), config.Login.Providers[providerID].CookieHashKey)
	assert.Equal(t, RedactedString("token-encryption-key-12345678910"), config.Login.TokenEncryption.SecretKey)
	assert.Equal(t, true, config.Login.Providers[providerID].Default)
}

func TestReadConfigWithEnvVarsNoSecretFile(t *testing.T) {
	t.Setenv("GATEWAY_LOGIN_PROVIDERS_RENKU_CLIENTSECRET", "env-var-secret")
	t.Setenv("GATEWAY_LOGIN_TOKENENCRYPTION_SECRETKEY", "token-encryption-key-12345678910")
	providerID := "renku"
	ch := NewConfigHandler()
	config, err := ch.Config()
	require.NoError(t, err)
	slog.Info("configuration data", "config", config)
	assert.NotEqual(t, config, Config{})
	assert.Len(t, config.Login.Providers, 1)
	assert.Equal(t, "https://renkulab.io", config.Revproxy.RenkuBaseURL.String())
	assert.Equal(t, RedactedString("env-var-secret"), config.Login.Providers[providerID].ClientSecret)
	assert.Equal(t, RedactedString("token-encryption-key-12345678910"), config.Login.TokenEncryption.SecretKey)
	assert.Equal(t, true, config.Login.Providers[providerID].Default)
}

