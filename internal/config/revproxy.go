package config

import (
	"fmt"
	"net/url"
)

type RenkuServicesConfig struct {
	DataService *url.URL
	Keycloak    *url.URL
	UIServer    *url.URL
}

type RevproxyConfig struct {
	EnableInternalGitlab bool
	RenkuBaseURL         *url.URL
	ExternalGitlabURL    *url.URL
	K8sNamespace         string
	RenkuServices        RenkuServicesConfig
}

func (r *RevproxyConfig) Validate() error {
	if r.RenkuServices.DataService == nil {
		return fmt.Errorf("the proxy config is missing the url to the data service")
	}
	if r.RenkuServices.Keycloak == nil {
		return fmt.Errorf("the proxy config is missing the url to keycloak")
	}
	if r.RenkuServices.UIServer == nil {
		return fmt.Errorf("the proxy config is missing the url to ui-server")
	}
	if r.RenkuBaseURL == nil {
		return fmt.Errorf("the renkuBaseURL cannot be null or ''")
	}

	return nil
}
