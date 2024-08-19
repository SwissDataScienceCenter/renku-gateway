package db

import (
	"context"
	"testing"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Check that RedisAdapter implements SessionRepository.
// This test would fail to compile otherwise.
func TestRedisAdapterIsSessionRepository(t *testing.T) {
	rdb := RedisAdapter{}
	_ = models.SessionRepository(rdb)
}

func TestSetGetSession(t *testing.T) {
	ctx := context.Background()
	adapter := NewMockRedisAdapter()
	sm := sessions.NewSessionMaker()
	mySession, err := sm.NewSession()
	require.NoError(t, err)
	err = adapter.SetSession(ctx, mySession)
	require.NoError(t, err)
	session, err := adapter.GetSession(ctx, mySession.ID)
	require.NoError(t, err)
	assert.Equal(t, mySession, session)
}

func TestRemoveSession(t *testing.T) {
	ctx := context.Background()
	adapter := NewMockRedisAdapter()
	sm := sessions.NewSessionMaker()
	mySession, err := sm.NewSession()
	require.NoError(t, err)
	err = adapter.SetSession(ctx, mySession)
	assert.NoError(t, err)
	err = adapter.RemoveSession(ctx, mySession.ID)
	assert.NoError(t, err)
	_, err = adapter.GetSession(ctx, mySession.ID)
	assert.ErrorIs(t, err, gwerrors.ErrSessionNotFound)
}
