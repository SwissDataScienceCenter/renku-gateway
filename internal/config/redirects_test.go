package config

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getValidRedirectsConfig(t *testing.T) RedirectsStoreConfig {
	renkuBaseURL, err := url.Parse("https://renku.example.org")
	require.NoError(t, err)
	return RedirectsStoreConfig{
		Gitlab: GitlabRedirectsConfig{
			Enabled:        true,
			RenkuBaseURL:   renkuBaseURL,
			RedirectedHost: "gitlab.example.org",
		},
	}
}

func TestValidRedirectsConfig(t *testing.T) {
	config := getValidRedirectsConfig(t)

	err := config.Validate()

	assert.NoError(t, err)
}

func TestInvalidRedirectsConfig(t *testing.T) {
	config := getValidRedirectsConfig(t)
	config.Gitlab.RenkuBaseURL = nil

	err := config.Validate()

	assert.ErrorContains(t, err, "the redirects store is enabled but the config is missing the base url for Renku")
}
