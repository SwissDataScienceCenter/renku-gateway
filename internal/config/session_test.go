package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	config := SessionConfig{
		IdleSessionTTLSeconds: 180,
		MaxSessionTTLSeconds:  360,
	}

	err := config.Validate(Development)

	assert.NoError(t, err)
}

func TestInvalidIdleSessionTTLSeconds(t *testing.T) {
	config := SessionConfig{
		IdleSessionTTLSeconds: -60,
	}

	err := config.Validate(Development)

	assert.ErrorContains(t, err, "idle session TTL seconds (-60) needs to be greater than 0")
}

func TestInvalidMaxSessionTTLSeconds(t *testing.T) {
	config := SessionConfig{
		IdleSessionTTLSeconds: 360,
		MaxSessionTTLSeconds:  180,
	}

	err := config.Validate(Development)

	assert.ErrorContains(t, err, "max session TTL seconds (180) cannot be less than idle session TTL seconds (360)")
}

func TestInvalidUnsafeNoCookieHandler(t *testing.T) {
	config := SessionConfig{
		IdleSessionTTLSeconds: 180,
		MaxSessionTTLSeconds:  360,
		UnsafeNoCookieHandler: true,
	}

	err := config.Validate(Production)

	assert.ErrorContains(t, err, "a cookie handler needs to be configured in production")
}
