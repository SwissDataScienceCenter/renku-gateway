package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func getValidConfig(t *testing.T) Config {
	sessionConfig := getValidSessionConfig()
	revproxyConfig := getValidRevproxyConfig(t)
	loginConfig := getValidLoginConfig(t)
	redisConfig := getValidRedisConfig()
	return Config{
		RunningEnvironment: Production,
		Sessions:           sessionConfig,
		Revproxy:           revproxyConfig,
		Login:              loginConfig,
		Redis:              redisConfig,
	}
}

func TestValidConfig(t *testing.T) {
	config := getValidConfig(t)

	err := config.Validate()

	assert.NoError(t, err)
}

func TestInvalidSessionsConfig(t *testing.T) {
	config := getValidConfig(t)
	config.Sessions.IdleSessionTTLSeconds = 0

	err := config.Validate()

	assert.Error(t, err)
}

func TestInvalidLoginConfig(t *testing.T) {
	config := getValidConfig(t)
	config.Login.TokenEncryption.SecretKey = "invalid"

	err := config.Validate()

	assert.Error(t, err)
}

func TestInvalidRevproxyConfig(t *testing.T) {
	config := getValidConfig(t)
	config.Revproxy.RenkuServices.Notebooks = nil

	err := config.Validate()

	assert.Error(t, err)
}

func TestInvalidRedisConfig(t *testing.T) {
	config := getValidConfig(t)
	config.Redis.Type = "redis-mock"

	err := config.Validate()

	assert.Error(t, err)
}
