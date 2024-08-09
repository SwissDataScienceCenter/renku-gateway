package db

import (
	"testing"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

// Check that RedisAdapter implements TokenRepository.
// This test would fail to compile otherwise.
func TestRedisAdapterIsTokenRepository(t *testing.T) {
	rdb := RedisAdapter{}
	_ = models.TokenRepository(rdb)
}
