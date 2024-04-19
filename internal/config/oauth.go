package config

type OAuthClientsConfig struct {
	Applications map[string]OAuthApplicationConfig
}

type OAuthApplicationConfig struct {
	ClientID    string
	DisplayName string
}
