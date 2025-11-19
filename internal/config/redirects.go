package config

import (
	"fmt"
	"net/url"
)

type GitlabRedirectsConfig struct {
	Enabled         bool
	RenkuBaseURL    *url.URL
	RedirectedHost  string
	EntryTtlSeconds int
}
type RedirectsStoreConfig struct {
	Gitlab GitlabRedirectsConfig
}

func (r *RedirectsStoreConfig) Validate() error {
	if r.Gitlab.Enabled && r.Gitlab.RenkuBaseURL == nil {
		return fmt.Errorf("the redirects store is enabled but the config is missing the base url for Renku")
	}

	if r.Gitlab.EntryTtlSeconds <= 0 {
		r.Gitlab.EntryTtlSeconds = 60 * 5 // default to 5 minutes
	}

	return nil
}
