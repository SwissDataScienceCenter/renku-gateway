package db

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"io"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var compareOptions []cmp.Option = []cmp.Option{cmpopts.IgnoreUnexported(models.AuthToken{})}

// Check that RedisAdapter implements TokenRepository.
// This test would fail to compile otherwise.
func TestRedisAdapterIsTokenRepository(t *testing.T) {
	rdb := RedisAdapter{}
	_ = models.TokenRepository(rdb)
}

func TestSetGetRemoveAccessToken(t *testing.T) {
	ctx := context.Background()
	adapter := NewMockRedisAdapter()
	myAccessToken := models.AuthToken{
		ID:        "12345",
		Value:     "6789",
		ExpiresAt: time.Now().Add(time.Hour * 24),
		TokenURL:  "https://gitlab.com",
		Type:      models.AccessTokenType,
	}
	err := adapter.SetAccessToken(ctx, myAccessToken)
	assert.NoError(t, err)
	accessToken, err := adapter.GetAccessToken(ctx, myAccessToken.ID)
	assert.NoError(t, err)
	assert.Truef(
		t,
		cmp.Equal(myAccessToken, accessToken, compareOptions...),
		"The two values are not equal, diff is: %s\n",
		cmp.Diff(myAccessToken, accessToken, compareOptions...),
	)
}

func TestSetGetAccessTokenWithEncryption(t *testing.T) {
	ctx := context.Background()
	secretKey := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, secretKey)
	require.NoError(t, err)
	adapter := NewMockRedisAdapter(WithEncryption(string(secretKey)))
	myAccessToken := models.AuthToken{
		ID:        "12345",
		Value:     "6789",
		ExpiresAt: time.Now().Add(time.Hour * 24),
		TokenURL:  "https://gitlab.com",
		Type:      models.AccessTokenType,
	}
	err = adapter.SetAccessToken(ctx, myAccessToken)
	assert.NoError(t, err)
	accessToken, err := adapter.GetAccessToken(ctx, myAccessToken.ID)
	assert.NoError(t, err)
	assert.Truef(
		t,
		cmp.Equal(myAccessToken, accessToken, compareOptions...),
		"The two values are not equal, diff is: %s\n",
		cmp.Diff(myAccessToken, accessToken, compareOptions...),
	)
}
