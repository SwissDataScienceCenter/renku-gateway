package db

import (
	"testing"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
)

// Check that RedisAdapter implements SessionRepository.
// This test would fail to compile otherwise.
func TestRedisAdapterIsSessionRepository(t *testing.T) {
	rdb := RedisAdapter{}
	_ = sessions.SessionRepository(rdb)
}
