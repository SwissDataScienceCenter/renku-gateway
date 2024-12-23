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
		RenkuBaseURL:      renkuBaseURL,
		ExternalGitlabURL: externalGitlabURL,
		RenkuServices:     renkuServicesConfig,
	}
}

func getValidRenkuServicesConfig(t *testing.T) RenkuServicesConfig {
	kgURL, err := url.Parse("http://kg")
	require.NoError(t, err)
	webhookURL, err := url.Parse("http://kg")
	require.NoError(t, err)
	dataServiceURL, err := url.Parse("http://data-service")
	require.NoError(t, err)
	keycloakURL, err := url.Parse("http://keycloak")
	require.NoError(t, err)
	uiServerURL, err := url.Parse("http://ui")
	require.NoError(t, err)
	searchURL, err := url.Parse("http://ui")
	require.NoError(t, err)
	return RenkuServicesConfig{
		KG:          kgURL,
		Webhook:     webhookURL,
		DataService: dataServiceURL,
		Keycloak:    keycloakURL,
		UIServer:    uiServerURL,
		Search:      searchURL,
	}
}

func TestValidRevproxyConfig(t *testing.T) {
	config := getValidRevproxyConfig(t)

	err := config.Validate()

	assert.NoError(t, err)
}

func TestInvalidKGURL(t *testing.T) {
	config := getValidRevproxyConfig(t)
	config.RenkuServices.KG = nil

	err := config.Validate()

	assert.ErrorContains(t, err, "the proxy config is missing the url to the knowledge graph service")
}

func TestInvalidWebhookURL(t *testing.T) {
	config := getValidRevproxyConfig(t)
	config.RenkuServices.Webhook = nil

	err := config.Validate()

	assert.ErrorContains(t, err, "the proxy config is missing the url to the webhook service")
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

func TestInvalidSearchURL(t *testing.T) {
	config := getValidRevproxyConfig(t)
	config.RenkuServices.Search = nil

	err := config.Validate()

	assert.ErrorContains(t, err, "the proxy config is missing the url to search")
}

func TestInvalidCoreSvcConfig(t *testing.T) {
	config := getValidRevproxyConfig(t)
	config.RenkuServices.Core.ServiceNames = []string{"core-svc"}
	config.RenkuServices.Core.ServicePaths = []string{}

	err := config.Validate()

	assert.ErrorContains(t, err, "the number of core service names and paths do not match")
}
