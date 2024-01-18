package config

import "fmt"

type TokenEncryptionConfig struct {
	Enabled   bool
	SecretKey string
}

type LoginConfig struct {
	DefaultAppRedirectURL string
	EndpointsBasePath     string
	TokenEncryption       TokenEncryptionConfig
	Providers             map[string]OIDCClient
}

type OIDCClient struct {
	Default           bool
	Issuer            string
	ClientID          string
	ClientSecret      string
	Scopes            []string
	CallbackURI       string
	UsePKCE           bool
	CookieEncodingKey string
	CookieHashKey     string
	// NOTE: should only be used for testing, in production this has to be false/unset
	// without this there is no CSRF protection on the oauth callback endpoint
	UnsafeNoCookieHandler bool
}

func (c *LoginConfig) Validate() error {
	if c.TokenEncryption.Enabled && len([]byte(c.TokenEncryption.SecretKey)) != 32 {
		return fmt.Errorf(
			"token encryption key has to be 32 bytes long, the provided one is %d long",
			len([]byte(c.TokenEncryption.SecretKey)),
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
