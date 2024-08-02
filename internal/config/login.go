package config

import (
	"fmt"
	"log/slog"
	"net/url"
)

type TokenEncryptionConfig struct {
	Enabled   bool
	SecretKey RedactedString
}

type LoginConfig struct {
	RenkuBaseURL      *url.URL
	EndpointsBasePath string
	TokenEncryption   TokenEncryptionConfig
	Providers         map[string]OIDCClient
}

type OIDCClient struct {
	Default           bool
	Issuer            string
	ClientID          string
	ClientSecret      RedactedString
	Scopes            []string
	CallbackURI       string
	UsePKCE           bool
	CookieEncodingKey RedactedString
	CookieHashKey     RedactedString
	// NOTE: UnsafeNoCookieHandler should only be used for testing, in production this has to be false/unset
	// without this there is no CSRF protection on the oauth callback endpoint
	UnsafeNoCookieHandler bool
}

func (c *LoginConfig) Validate() error {
	slog.Info("login configuration info", "config", c)
	if c.TokenEncryption.Enabled && len(c.TokenEncryption.SecretKey) != 32 {
		return fmt.Errorf(
			"token encryption key has to be 32 bytes long, the provided one is %d long",
			len(c.TokenEncryption.SecretKey),
		)
	}
	return nil
}

func (c LoginConfig) DefaultProviderIDs() []string {
	output := []string{}
	for id, provider := range c.Providers {
		if provider.Default {
			output = append(output, id)
		}
	}
	return output
}
