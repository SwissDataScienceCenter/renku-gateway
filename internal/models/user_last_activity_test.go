package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserLastActivityTouch(t *testing.T) {
	userLastActivity := UserLastActivity{}
	userLastActivity.Touch()
	assert.False(t, userLastActivity.LastActivity.IsZero())
	assert.True(t, userLastActivity.ExpiresAt.After(userLastActivity.LastActivity))
}
