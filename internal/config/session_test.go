package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func getValidSessionConfig() SessionConfig {
	return SessionConfig{
		IdleSessionTTLSeconds: 14400,
		MaxSessionTTLSeconds:  86400,
	}
}

func TestValidSessionConfig(t *testing.T) {
	config := getValidSessionConfig()

	err := config.Validate(Production)

	assert.NoError(t, err)
}

func TestInvalidIdleSessionTTLSeconds(t *testing.T) {
	config := getValidSessionConfig()
	config.IdleSessionTTLSeconds = -60

	err := config.Validate(Production)

	assert.ErrorContains(t, err, "idle session TTL seconds (-60) needs to be greater than 0")
}

func TestInvalidMaxSessionTTLSeconds(t *testing.T) {
	config := getValidSessionConfig()
	config.MaxSessionTTLSeconds = 600

	err := config.Validate(Production)

	assert.ErrorContains(t, err, "max session TTL seconds (600) cannot be less than idle session TTL seconds (14400)")
}

func TestInvalidUnsafeNoCookieHandler(t *testing.T) {
	config := getValidSessionConfig()
	config.UnsafeNoCookieHandler = true

	err := config.Validate(Production)

	assert.ErrorContains(t, err, "a cookie handler needs to be configured in production")
}
