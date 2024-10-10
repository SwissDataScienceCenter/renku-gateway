package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func getValidRedisConfig() RedisConfig {
	return RedisConfig{
		Type: "redis",
	}
}

func TestValidRedisConfig(t *testing.T) {
	config := getValidRedisConfig()

	err := config.Validate(Production)

	assert.NoError(t, err)
}

func TestInvalidRedisType(t *testing.T) {
	config := getValidRedisConfig()
	config.Type = "redis-mock"

	err := config.Validate(Production)

	assert.ErrorContains(t, err, "redis type cannot be \"redis-mock\" in production")
}
