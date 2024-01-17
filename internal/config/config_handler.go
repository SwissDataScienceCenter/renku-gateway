package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"reflect"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type ConfigHandler struct {
	mainViper   *viper.Viper
	secretViper *viper.Viper
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
// The secret file will always overwrite anything in the non-secret / regular file.
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
	return &ConfigHandler{secretViper: secret, mainViper: main, lock: &sync.Mutex{}}
}

func (c *ConfigHandler) merge() error {
	fname := c.secretViper.ConfigFileUsed()
	if fname == "" {
		return fmt.Errorf("cannot find secret config")
	}
	fp, err := os.Open(fname)
	defer fp.Close()
	if err != nil {
		return err
	}
	err = c.mainViper.MergeConfig(fp)
	if err != nil {
		return err
	}
	return nil
}

func (c *ConfigHandler) getConfig() (Config, error) {
	var output Config
	err := c.mainViper.ReadInConfig()
	if err != nil {
		return Config{}, err
	}
	err = c.secretViper.ReadInConfig()
	if err != nil {
		return Config{}, err
	}
	err = c.merge()
	err = c.mainViper.Unmarshal(
		&output,
		viper.DecodeHook(
			mapstructure.ComposeDecodeHookFunc(
				parseStringAsURL(),
			),
		),
	)
	if err != nil {
		return Config{}, err
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
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
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
