package db

import (
	"testing"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

// Check that RedisAdapter implements SessionRepository.
// This test would fail to compile otherwise.
func TestRedisAdapterIsSessionRepository(t *testing.T) {
	rdb := RedisAdapter{}
	_ = models.SessionRepository(rdb)
}
