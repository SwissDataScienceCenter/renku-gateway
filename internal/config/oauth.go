package config

import "strings"

type OAuthClientsConfig struct {
	Applications map[string]OAuthApplicationConfig
}

type OAuthApplicationConfig struct {
	ClientID    string
	DisplayName string
}

func (c *OAuthClientsConfig) Validate() error {
	apps := map[string]OAuthApplicationConfig{}
	for id, provider := range c.Applications {
		idWithDots := strings.Replace(id, ":", ".", -1)
		apps[idWithDots] = provider
	}
	c.Applications = apps
	return nil
}
