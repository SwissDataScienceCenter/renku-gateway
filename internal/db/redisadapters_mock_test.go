package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHGetAll(t *testing.T) {
	ctx := context.Background()
	store := MockRedisClient{map[string]interface{}{}}
	res := store.HGetAll(ctx, "test")
	val, err := res.Result()
	require.NoError(t, err)
	assert.Equal(t, 0, len(val))
}

func TestHSetDel(t *testing.T) {
	ctx := context.Background()
	store := MockRedisClient{map[string]interface{}{}}
	res1 := store.HSet(ctx, "test", "f1", "v1", "f2", "v2")
	_, err := res1.Result()
	require.NoError(t, err)
	res2 := store.HGetAll(ctx, "test")
	val, err := res2.Result()
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"f1": "v1", "f2": "v2"}, val)
	res3 := store.Del(ctx, "test")
	_, err = res3.Result()
	require.NoError(t, err)
	res4 := store.HGetAll(ctx, "test")
	val, err = res4.Result()
	require.NoError(t, err)
	assert.Equal(t, 0, len(val))
}
