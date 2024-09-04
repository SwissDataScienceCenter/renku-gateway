package db

import (
	"context"
	"encoding"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Implements the LimitedRedis client struct
// Only suitable for testing
// The value set for the IntCmd or similar results is always 1 regardless of how many records were affected
// Contexts are completely ignored
type MockRedisClient struct {
	store map[string]any
}

type MockRedisAdapterOption func(r *RedisAdapter)

func WithEncryption(key string) MockRedisAdapterOption {
	return func(r *RedisAdapter) {
		enc, err := NewGCMEncryptor(key)
		if err != nil {
			log.Fatalln(err)
		}
		r.encryptor = enc
	}
}

func NewMockRedisAdapter(options ...MockRedisAdapterOption) RedisAdapter {
	store := MockRedisClient{map[string]any{}}
	db := RedisAdapter{rdb: &store}
	for _, opt := range options {
		opt(&db)
	}
	return db
}

func convertValuesToMap(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return map[string]any{}, fmt.Errorf("number of provided values must be even")
	}
	output := map[string]any{}
	for i := 0; i < len(values); i += 2 {
		key := values[i].(string)
		val := values[i+1]
		output[key] = val
	}
	return output, nil
}

func (m *MockRedisClient) HSet(_ context.Context, key string, values ...any) *redis.IntCmd {
	res := redis.IntCmd{}
	val, err := convertValuesToMap(values...)
	if err != nil {
		res.SetErr(err)
	}

	m.store[key] = val
	res.SetVal(1)
	return &res
}

func (m *MockRedisClient) Del(_ context.Context, keys ...string) *redis.IntCmd {
	for _, k := range keys {
		delete(m.store, k)
	}
	res := redis.IntCmd{}
	res.SetVal(1)
	return &res
}

func (m *MockRedisClient) HGetAll(_ context.Context, key string) *redis.MapStringStringCmd {
	val, found := m.store[key]
	res := redis.MapStringStringCmd{}
	res.SetVal(map[string]string{})
	res.SetErr(nil)
	if !found {
		return &res
	}
	valMap1 := val.(map[string]any)
	valMap2 := map[string]string{}
	for k, v := range valMap1 {
		valString, ok := v.(string)
		if !ok {
			encodable := v.(encoding.TextMarshaler)
			valString, err := encodable.MarshalText()
			if err != nil {
				res.SetErr(err)
				return &res
			}
			valMap2[k] = string(valString)
		} else {
			valMap2[k] = valString
		}
	}
	res.SetVal(valMap2)
	return &res
}

func (m *MockRedisClient) ExpireAt(ctx context.Context, key string, tm time.Time) *redis.BoolCmd {
	output := redis.BoolCmd{}
	output.SetVal(true)
	return &output
}

func (m *MockRedisClient) Persist(ctx context.Context, key string) *redis.BoolCmd {
	output := redis.BoolCmd{}
	output.SetVal(true)
	return &output
}
