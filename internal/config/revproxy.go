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
	Search      *url.URL
}

type RevproxyConfig struct {
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
	if r.RenkuServices.KG == nil {
		return fmt.Errorf("the proxy config is missing the url to the knowledge graph service")
	}
	if r.RenkuServices.Webhook == nil {
		return fmt.Errorf("the proxy config is missing the url to the webhook service")
	}
	if r.RenkuServices.DataService == nil {
		return fmt.Errorf("the proxy config is missing the url to the data service")
	}
	if r.RenkuServices.Keycloak == nil {
		return fmt.Errorf("the proxy config is missing the url to keycloak")
	}
	if r.RenkuServices.UIServer == nil {
		return fmt.Errorf("the proxy config is missing the url to ui-server")
	}
	if r.RenkuServices.Search == nil {
		return fmt.Errorf("the proxy config is missing the url to search")
	}
	if len(r.RenkuServices.Core.ServiceNames) != len(r.RenkuServices.Core.ServicePaths) {
		return fmt.Errorf("the number of core service names and paths do not match")
	}
	return nil
}
