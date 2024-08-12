package db

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// LimitedRedisClient is the limited set of functionality expected from the redis client in this adapter.
// This allows for easy mocking and swapping of the client. The universal redis client interface is way too big.
type LimitedRedisClient interface {
	// General commands

	// EXPIREAT key unix-time-seconds
	ExpireAt(ctx context.Context, key string, tm time.Time) *redis.BoolCmd
	// DEL key [key ...]
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	// PERSIST key
	Persist(ctx context.Context, key string) *redis.BoolCmd

	// Hash commands

	// HGETALL key
	HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd
	// HSET key field value [field value ...]
	HSet(ctx context.Context, key string, values ...any) *redis.IntCmd
}
