package config

import (
	"fmt"
	"net/url"
)

type RedirectsStoreConfig struct {
	Enabled        bool
	RenkuBaseURL   *url.URL
	RedirectedHost string
}

func (r *RedirectsStoreConfig) Validate() error {
	if r.Enabled && r.RenkuBaseURL == nil {
		return fmt.Errorf("the redirects store is enabled but the config is missing the base url for Renku")
	}

	return nil
}
