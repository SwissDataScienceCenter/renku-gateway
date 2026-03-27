package db

import (
	"testing"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Check that RedisAdapter implements TokenRepository.
// This test would fail to compile otherwise.
func TestRedisAdapterIsTokenRefreshRepository(t *testing.T) {
	rdb := RedisAdapter{}
	_ = models.TokenRefreshRepository(rdb)
}

func TestGetExpiringRefreshTokenIDs(t *testing.T) {
	ctx := t.Context()
	adapter := NewMockRedisAdapter()
	// Add a token which expires in 30 minutes
	myRefreshToken := models.AuthToken{
		ID:        "12345",
		Value:     "6789",
		ExpiresAt: time.Now().Add(30 * time.Minute),
		TokenURL:  "https://renku.ch",
		Type:      models.RefreshTokenType,
	}
	err := adapter.SetRefreshToken(ctx, myRefreshToken)
	require.NoError(t, err)
	// Also add a token which expires in 24 hours
	err = adapter.SetRefreshToken(ctx, models.AuthToken{
		ID:        "54321",
		Value:     "9876",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		TokenURL:  "https://renku.ch",
		Type:      models.RefreshTokenType,
	})
	require.NoError(t, err)
	// Get refresh tokens which expires between [now, now + 1 hour]
	tokenIDs, err := adapter.GetExpiringRefreshTokenIDs(ctx, time.Now(), time.Now().Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, []string{myRefreshToken.ID}, tokenIDs)
}
