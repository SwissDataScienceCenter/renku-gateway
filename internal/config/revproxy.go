package config

import (
	"fmt"
	"net/url"
)

type RenkuServicesConfig struct {
	KG          *url.URL
	Webhook     *url.URL
	Core        CoreSvcConfig
	DataService *url.URL
	Keycloak    *url.URL
	UIServer    *url.URL
}

type RevproxyConfig struct {
	EnableV1Services  bool
	RenkuBaseURL      *url.URL
	ExternalGitlabURL *url.URL
	K8sNamespace      string
	RenkuServices     RenkuServicesConfig
}

type CoreSvcConfig struct {
	ServiceNames []string
	ServicePaths []string
	Sticky       bool
}

func (r *RevproxyConfig) Validate() error {
	// Check v2 services first
	if r.RenkuServices.DataService == nil {
		return fmt.Errorf("the proxy config is missing the url to the data service")
	}
	if r.RenkuServices.Keycloak == nil {
		return fmt.Errorf("the proxy config is missing the url to keycloak")
	}
	if r.RenkuServices.UIServer == nil {
		return fmt.Errorf("the proxy config is missing the url to ui-server")
	}

	// Check v1 services if needed
	if r.EnableV1Services {
		if r.RenkuServices.KG == nil {
			return fmt.Errorf("the proxy config is missing the url to the knowledge graph service")
		}
		if r.RenkuServices.Webhook == nil {
			return fmt.Errorf("the proxy config is missing the url to the webhook service")
		}
		if len(r.RenkuServices.Core.ServiceNames) != len(r.RenkuServices.Core.ServicePaths) {
			return fmt.Errorf("the number of core service names and paths do not match")
		}
	}

	return nil
}
