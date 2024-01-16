package config

import (
	"net/url"
)

type renkuServicesConfig struct {
	Notebooks        *url.URL
	KG               *url.URL
	Webhook          *url.URL
	CoreServiceNames []string
	CoreServicePaths []string
	Auth             *url.URL
	Crc              *url.URL
}

type metricsConfig struct {
	Enabled bool
	Port    int
}

type RevproxyConfig struct {
	RenkuBaseURL      *url.URL
	ExternalGitlabURL *url.URL
	K8sNamespace      string
	RenkuServices     renkuServicesConfig
}
