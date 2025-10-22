package config

import (
	"fmt"
	"net/url"
)

type RedirectsStoreConfig struct {
	RenkuBaseURL   *url.URL
	RedirectedHost string
}

func (r *RedirectsStoreConfig) Validate() error {
	if r.RenkuBaseURL == nil {
		return fmt.Errorf("the redirects store config is missing the base url for Renku")
	}

	return nil
}
