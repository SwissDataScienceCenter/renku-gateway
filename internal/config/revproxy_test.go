package config

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getValidRevproxyConfig(t *testing.T) RevproxyConfig {
	renkuBaseURL, err := url.Parse("https://renku.example.org")
	require.NoError(t, err)
	externalGitlabURL, err := url.Parse("https://gitlab.example.org")
	require.NoError(t, err)
	renkuServicesConfig := getValidRenkuServicesConfig(t)
	return RevproxyConfig{
		EnableInternalGitlab: true,
		RenkuBaseURL:         renkuBaseURL,
		ExternalGitlabURL:    externalGitlabURL,
		RenkuServices:        renkuServicesConfig,
	}
}

func getValidRenkuServicesConfig(t *testing.T) RenkuServicesConfig {
	dataServiceURL, err := url.Parse("http://data-service")
	require.NoError(t, err)
	keycloakURL, err := url.Parse("http://keycloak")
	require.NoError(t, err)
	uiServerURL, err := url.Parse("http://ui")
	require.NoError(t, err)
	return RenkuServicesConfig{
		DataService: dataServiceURL,
		Keycloak:    keycloakURL,
		UIServer:    uiServerURL,
	}
}

func TestValidRevproxyConfig(t *testing.T) {
	config := getValidRevproxyConfig(t)

	err := config.Validate()

	assert.NoError(t, err)
}

func TestInvalidDataServiceURL(t *testing.T) {
	config := getValidRevproxyConfig(t)
	config.RenkuServices.DataService = nil

	err := config.Validate()

	assert.ErrorContains(t, err, "the proxy config is missing the url to the data service")
}

func TestInvalidKeycloakURL(t *testing.T) {
	config := getValidRevproxyConfig(t)
	config.RenkuServices.Keycloak = nil

	err := config.Validate()

	assert.ErrorContains(t, err, "the proxy config is missing the url to keycloak")
}

func TestInvalidUIServerURL(t *testing.T) {
	config := getValidRevproxyConfig(t)
	config.RenkuServices.UIServer = nil

	err := config.Validate()

	assert.ErrorContains(t, err, "the proxy config is missing the url to ui-server")
}
