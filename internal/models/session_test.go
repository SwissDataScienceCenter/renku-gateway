package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionNotExpired(t *testing.T) {
	session := Session{
		ExpiresAt: time.Now().Add(time.Duration(5) * time.Minute),
	}
	assert.False(t, session.Expired())
}

func TestSessionExpired(t *testing.T) {
	session := Session{
		ExpiresAt: time.Now().Add(-time.Duration(5) * time.Minute),
	}
	assert.True(t, session.Expired())
}

func TestSessionTouch(t *testing.T) {
	session := Session{
		CreatedAt:      time.Now(),
		IdleTTLSeconds: 100,
		MaxTTLSeconds:  300,
	}
	session.Touch()
	assert.False(t, session.Expired())
	assert.True(t, session.ExpiresAt.After(session.CreatedAt))
	assert.True(t, session.ExpiresAt.Before(session.CreatedAt.Add(session.MaxTTL())))
}

func TestSessionTouchCloseToMaxTTL(t *testing.T) {
	session := Session{
		CreatedAt:      time.Now().Add(-time.Duration(200) * time.Second),
		IdleTTLSeconds: 100,
		MaxTTLSeconds:  300,
	}
	session.Touch()
	assert.False(t, session.Expired())
	assert.True(t, session.ExpiresAt.After(session.CreatedAt))
	assert.Equal(t, session.ExpiresAt.UTC(), session.CreatedAt.Add(session.MaxTTL()).UTC())
}

func TestSessionTouchNoExpiry(t *testing.T) {
	session := Session{
		CreatedAt: time.Now(),
	}
	session.Touch()
	assert.False(t, session.Expired())
	assert.True(t, session.ExpiresAt.IsZero())
}

func TestSessionLoginState(t *testing.T) {
	session := Session{
		CreatedAt: time.Now(),
	}
	err := session.GenerateLoginState()
	require.NoError(t, err)
	assert.NotEmpty(t, session.LoginState)
	state := session.LoginState
	err = session.GenerateLoginState()
	require.NoError(t, err)
	assert.NotEmpty(t, session.LoginState)
	assert.NotEqual(t, state, session.LoginState)
}
