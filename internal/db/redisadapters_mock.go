package db

import (
	"context"
	"encoding"
	"fmt"
	"log"
	"sort"

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

func (m *MockRedisClient) ZAdd(_ context.Context, key string, members ...redis.Z) *redis.IntCmd {
	_, found := m.store[key]
	if !found {
		m.store[key] = []redis.Z{}
	}
	newMembers := append(m.store[key].([]redis.Z), members...)
	sort.Slice(newMembers, func(i, j int) bool { return newMembers[i].Score < newMembers[j].Score })
	m.store[key] = newMembers
	res := redis.IntCmd{}
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

func (m *MockRedisClient) ZRem(_ context.Context, key string, members ...any) *redis.IntCmd {
	val, found := m.store[key]
	res := redis.IntCmd{}
	if !found {
		res.SetVal(0)
	}
	valZ := val.([]redis.Z)
	newValZ := []redis.Z{}
	for _, z := range valZ {
		var removeElem = false
		for _, member := range members {
			removeElem = removeElem || (z.Member == member)
		}
		if !removeElem {
			newValZ = append(newValZ, z)
		}
	}

	m.store[key] = newValZ
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

func (m *MockRedisClient) ZRangeArgsWithScores(_ context.Context, zrange redis.ZRangeArgs) *redis.ZSliceCmd {
	valRaw, found := m.store[zrange.Key]
	if !found {
		return &redis.ZSliceCmd{}
	}
	val := valRaw.([]redis.Z)
	res := []redis.Z{}
	for _, ival := range val {
		if ival.Score <= zrange.Stop.(float64) && ival.Score >= zrange.Start.(float64) {
			res = append(res, ival)
		}
	}
	output := redis.ZSliceCmd{}
	output.SetVal(res)
	return &output
}
