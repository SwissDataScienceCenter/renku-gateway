package db

import (
	"context"
	"testing"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Check that RedisAdapter implements UserLastActivityRepository.
// This test would fail to compile otherwise.
func TestRedisAdapterIsUserLastActivityRepository(t *testing.T) {
	rdb := RedisAdapter{}
	_ = models.UserLastActivityRepository(rdb)
}

func TestSetGetUserLastActivity(t *testing.T) {
	ctx := context.Background()
	adapter := NewMockRedisAdapter()
	myUserLastActivity := models.UserLastActivity{
		UserID: "jane-doe",
	}
	err := adapter.SetUserLastActivity(ctx, myUserLastActivity)
	require.NoError(t, err)
	userLastActivity, err := adapter.GetUserLastActivity(ctx, myUserLastActivity.UserID)
	require.NoError(t, err)
	assert.Equal(t, myUserLastActivity, userLastActivity)
}
