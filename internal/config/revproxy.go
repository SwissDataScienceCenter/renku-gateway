package config

import (
	"net/url"
)

type RenkuServicesConfig struct {
	Notebooks        *url.URL
	KG               *url.URL
	Webhook          *url.URL
	Core             CoreSvcConfig
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
	RenkuServices     RenkuServicesConfig
}

type CoreSvcConfig struct {
	ServiceNames []string
	ServicePaths []string
	Sticky bool
}

