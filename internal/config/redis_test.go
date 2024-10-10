package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func getValidRedisConfig(t *testing.T) RedisConfig {
	return RedisConfig{
		Type: "redis",
	}
}

func TestValidRedisConfig(t *testing.T) {
	config := getValidRedisConfig(t)

	err := config.Validate(Production)

	assert.NoError(t, err)
}

func TestInvalidRedisType(t *testing.T) {
	config := getValidRedisConfig(t)
	config.Type = "redis-mock"

	err := config.Validate(Production)

	assert.ErrorContains(t, err, "redis type cannot be \"redis-mock\" in production")
}
