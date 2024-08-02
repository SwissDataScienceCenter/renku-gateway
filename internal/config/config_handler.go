package config

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type ConfigHandler struct {
	mainViper   *viper.Viper
	secretViper *viper.Viper
	envPrefix   string
	lock        *sync.Mutex
}

func (c *ConfigHandler) HandleChanges(callback func(Config, error)) {
	c.mainViper.OnConfigChange(func(e fsnotify.Event) {
		slog.Info("main config file changed", "path", e.Name)
		callback(c.Config())
	})
	c.secretViper.OnConfigChange(func(e fsnotify.Event) {
		slog.Info("secret config file changed", "path", e.Name)
		callback(c.Config())
	})
}

// Creates a configuration handler that reads the configuration files, merges them and can watch
// them for changes. Please note that the merges replace whole arrays - they do not merge arrays.
// The secret file will always overwrite anything in the non-secret / regular file. And any environment
// variables will always rewrite stuff in the secret config, so the order of preference from most
// preferred to least is environment variables, secret config, non-secret config.
func NewConfigHandler() *ConfigHandler {
	main := viper.New()
	main.SetConfigType("yaml")
	main.SetConfigName("config")
	secret := viper.New()
	secret.SetConfigType("yaml")
	secret.SetConfigName("secret_config")
	// Viper will look through the list of paths and use the first one where there is a file
	// so the path specified in the env variable will always take precedence over the rest
	configPaths := []string{}
	configPathEnv := os.Getenv("CONFIG_LOCATION")
	if configPathEnv != "" {
		configPaths = append(configPaths, configPathEnv)
	}
	configPaths = append(configPaths, "/etc/gateway", ".")
	for _, path := range configPaths {
		main.AddConfigPath(path)
		secret.AddConfigPath(path)
	}
	// Set the defaults to the main config
	var def map[string]any
	err := mapstructure.Decode(Config{}, &def)
	if err != nil {
		// NOTE: the error here can include the whole config file with sensitive fields shown in the logs
		slog.Error("could not decode default configuration struct into map[string]any")
		os.Exit(1)
	}
	main.MergeConfigMap(def)
	return &ConfigHandler{secretViper: secret, mainViper: main, lock: &sync.Mutex{}, envPrefix: "GATEWAY_"}
}

func (c *ConfigHandler) getConfig() (Config, error) {
	// NOTE: returning the error is avoided on purpose in most cases here because the error could
	// contain sensitive data from the config file or data that is being read in
	err := c.mainViper.MergeInConfig()
	if err != nil {
		return Config{}, fmt.Errorf("could not read the main configuration file")
	}
	// read secret config
	err = c.secretViper.ReadInConfig()
	if err != nil {
		switch err.(type) {
		default:
			return Config{}, fmt.Errorf("reading in the secret config failed")
		case viper.ConfigFileNotFoundError:
			slog.Info("could not find any secret config files - only the public file and environment variables will be used")
		}
	}
	err = c.mainViper.MergeConfigMap(c.secretViper.AllSettings())
	if err != nil {
		return Config{}, fmt.Errorf("could not merge the secret file config")
	}
	// read environment variables
	envVarsFiltered := []string{}
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, c.envPrefix) {
			continue
		}
		envVarsFiltered = append(envVarsFiltered, kv)
	}
	envBuf := bytes.NewBuffer([]byte(strings.Join(envVarsFiltered, "\n")))
	envViper := viper.New()
	envViper.SetConfigType("env")
	err = envViper.ReadConfig(envBuf)
	if err != nil {
		return Config{}, fmt.Errorf("could not read the environment variables into a config")
	}
	envData := envViper.AllSettings()
	prefix := strings.ToLower(c.envPrefix)
	for key, val := range envData {
		dataKey := strings.TrimPrefix(key, prefix)
		dataKey = strings.ReplaceAll(dataKey, "_", ".")
		c.mainViper.Set(dataKey, val)
	}
	// unmarshal and return
	var output Config
	dh := viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			parseStringAsURL(),
		),
	)
	err = c.mainViper.Unmarshal(&output, dh)
	if err != nil {
		return Config{}, fmt.Errorf("cannot unmarshal the combined config into a struct")
	}
	// NOTE: websockets proxying does not work if the port of the uiserver is not explicitly set
	if output.Revproxy.RenkuServices.UIServer != nil && output.Revproxy.RenkuServices.UIServer.Port() == "" {
		if output.Revproxy.RenkuServices.UIServer.Scheme == "http" {
			output.Revproxy.RenkuServices.UIServer.Host += ":80"
		} else if output.Revproxy.RenkuServices.UIServer.Scheme == "https" {
			output.Revproxy.RenkuServices.UIServer.Host += ":443"
		}
	}
	return output, nil
}

func (c *ConfigHandler) Config() (Config, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.getConfig()
}

func (c *ConfigHandler) Watch() {
	c.mainViper.WatchConfig()
	c.secretViper.WatchConfig()
}

func parseStringAsURL() mapstructure.DecodeHookFuncType {
	return func(f reflect.Type, t reflect.Type, data any) (interface{}, error) {
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
