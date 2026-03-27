package db

import (
	"math"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockRedisClient() *MockRedisClient {
	return &MockRedisClient{store: map[string]any{}}
}

func TestHGetAll(t *testing.T) {
	ctx := t.Context()
	rdb := newMockRedisClient()
	res := rdb.HGetAll(ctx, "test")
	val, err := res.Result()
	require.NoError(t, err)
	assert.Equal(t, 0, len(val))
}

func TestHSetDel(t *testing.T) {
	ctx := t.Context()
	rdb := newMockRedisClient()
	res1 := rdb.HSet(ctx, "test", "f1", "v1", "f2", "v2")
	_, err := res1.Result()
	require.NoError(t, err)
	res2 := rdb.HGetAll(ctx, "test")
	val, err := res2.Result()
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"f1": "v1", "f2": "v2"}, val)
	res3 := rdb.Del(ctx, "test")
	_, err = res3.Result()
	require.NoError(t, err)
	res4 := rdb.HGetAll(ctx, "test")
	val, err = res4.Result()
	require.NoError(t, err)
	assert.Equal(t, 0, len(val))
}

func TestZAddZRange(t *testing.T) {
	ctx := t.Context()
	rdb := newMockRedisClient()
	res1 := rdb.ZAdd(ctx, "zset", redis.Z{Score: 1, Member: "m1"}, redis.Z{Score: 1, Member: "m2"})
	res1Val, err := res1.Result()
	require.NoError(t, err)
	assert.Equal(t, int64(2), res1Val)
	res2 := rdb.ZAdd(ctx, "zset", redis.Z{Score: 2, Member: "m2"}, redis.Z{Score: 3, Member: "m3"})
	res2Val, err := res2.Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), res2Val)
	res3 := rdb.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     "zset",
		ByScore: true,
		Start:   1,
		Stop:    2,
	})
	res3Val, err := res3.Result()
	require.NoError(t, err)
	assert.Equal(t, []string{"m1", "m2"}, res3Val)
	res4 := rdb.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     "zset",
		ByScore: true,
		Start:   "(1",
		Stop:    "+inf",
	})
	res4Val, err := res4.Result()
	require.NoError(t, err)
	assert.Equal(t, []string{"m2", "m3"}, res4Val)
}

func TestParseZStartStop(t *testing.T) {
	var x any
	x = "-inf"
	value, _, err := parseZStartStop(x)
	require.NoError(t, err)
	assert.Equal(t, math.Inf(-1), value)
	x = "inf"
	value, _, err = parseZStartStop(x)
	require.NoError(t, err)
	assert.Equal(t, math.Inf(1), value)
	x = "+inf"
	value, _, err = parseZStartStop(x)
	require.NoError(t, err)
	assert.Equal(t, math.Inf(1), value)
	x = 10
	value, include, err := parseZStartStop(x)
	require.NoError(t, err)
	assert.Equal(t, float64(10), value)
	assert.True(t, include)
	x = "(10"
	value, include, err = parseZStartStop(x)
	require.NoError(t, err)
	assert.Equal(t, float64(10), value)
	assert.False(t, include)
	x = float64(12.34)
	value, include, err = parseZStartStop(x)
	require.NoError(t, err)
	assert.Equal(t, float64(12.34), value)
	assert.True(t, include)
	x = "(12.34"
	value, include, err = parseZStartStop(x)
	require.NoError(t, err)
	assert.Equal(t, float64(12.34), value)
	assert.False(t, include)
}
