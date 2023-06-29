package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type renkuServicesConfig struct {
	Notebooks        *url.URL `mapstructure:"renku_services_notebooks"`
	KG               *url.URL `mapstructure:"renku_services_kg"`
	Webhook          *url.URL `mapstructure:"renku_services_webhook"`
	CoreServiceNames []string `mapstructure:"renku_services_core_service_names"`
	CoreServicePaths []string `mapstructure:"renku_services_core_service_paths"`
	Auth             *url.URL `mapstructure:"renku_services_auth"`
	Crc      *url.URL `mapstructure:"renku_services_crc"`
}

type metricsConfig struct {
	Enabled bool `mapstructure:"metrics_enabled"`
	Port    int  `mapstructure:"metrics_port"`
}

type rateLimits struct {
	Enabled bool    `mapstructure:"rate_limits_enabled"`
	Rate    float64 `mapstructure:"rate_limits_average"`
	Burst   int     `mapstructure:"rate_limits_burst"`
}

type sentryConfig struct {
	Enabled     bool    `mapstructure:"sentry_enabled"`
	Dsn         string  `mapstructure:"sentry_dsn"`
	Environment string  `mapstructure:"sentry_environment"`
	SampleRate  float64 `mapstructure:"sentry_sample_rate"`
}

type revProxyConfig struct {
	RenkuBaseURL      *url.URL            `mapstructure:"renku_base_url"`
	AllowOrigin       []string            `mapstructure:"allow_origin"`
	ExternalGitlabURL *url.URL            `mapstructure:"external_gitlab_url"`
	RenkuServices     renkuServicesConfig `mapstructure:",squash"`
	Metrics           metricsConfig       `mapstructure:",squash"`
	RateLimits        rateLimits          `mapstructure:",squash"`
	Namespace         string              `mapstructure:"namespace"`
	Debug             bool
	Port              int
	Sentry            sentryConfig `mapstructure:",squash"`
}

func parseStringAsURL() mapstructure.DecodeHookFuncType {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		// Check that the data is string
		if f.Kind() != reflect.String {
			return data, nil
		}

		// Check that the target type is our custom type
		if t != reflect.TypeOf(url.URL{}) {
			return data, nil
		}

		// Return the parsed value
		dataStr, ok := data.(string)
		if !ok {
			return nil, fmt.Errorf("cannot cast URL value to string")
		}
		if dataStr == "" {
			return nil, fmt.Errorf("empty values are not allowed for URLs")
		}
		url, err := url.Parse(dataStr)
		if err != nil {
			return nil, err
		}
		return url, nil
	}
}

func getConfig() revProxyConfig {
	var config revProxyConfig
	prefix := "revproxy"
	viper.SetEnvPrefix(prefix)
	viper.AutomaticEnv()
	viper.AllowEmptyEnv(false)
	envKeysMap := &map[string]interface{}{}
	if err := mapstructure.Decode(config, &envKeysMap); err != nil {
		log.Fatal(err)
	}
	for k := range *envKeysMap {
		if _, ok := os.LookupEnv(strings.ToUpper(prefix) + "_" + strings.ToUpper(k)); !ok {
			log.Fatalf("Environment variable %s is not defined\n", strings.ToUpper(prefix)+"_"+strings.ToUpper(k))
		}
		if bindErr := viper.BindEnv(k); bindErr != nil {
			log.Fatal(bindErr)
		}
	}
	err := viper.Unmarshal(
		&config,
		viper.DecodeHook(
			mapstructure.ComposeDecodeHookFunc(
				parseStringAsURL(),
            	mapstructure.StringToSliceHookFunc(","),
			),
		),
	)
	if err != nil {
		log.Fatalf("unable to decode config into struct, %v\n", err)
	}
	return config
}

// AddQueryParams makes a copy of the provided URL, adds the query parameters
// and returns a url with the added parameters. The original URL is left unchanged.
func AddQueryParams(url *url.URL, params map[string]string) *url.URL {
	newURL := *url
	query := newURL.Query()
	for k, v := range params {
		query.Add(k, v)
	}
	newURL.RawQuery = query.Encode()
	return &newURL
}
