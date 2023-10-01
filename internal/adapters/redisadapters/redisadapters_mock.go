package redisadapters

import (
	"encoding"
	"fmt"

	"github.com/go-redis/redis/v9"
	"golang.org/x/net/context"
)

// Implements the LimitedRedis client struct
// Only suitable for testing
// The value set for the IntCmd or similar results is always 1 regardless of how many records were affected
// Contexts are completely ignored
type MockRedisClient struct {
	store map[string]interface{}
}

func NewMockRedisAdapter() RedisAdapter {
	store := MockRedisClient{map[string]interface{}{}}
	return RedisAdapter{rdb: &store}
}

func convertValuesToMap(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return map[string]interface{}{}, fmt.Errorf("number of provided values must be even")
	}
	output := map[string]interface{}{}
	for i := 0; i < len(values); i += 2 {
		key := values[i].(string)
		val := values[i+1]
		output[key] = val
	}
	return output, nil
}

func (m *MockRedisClient) HSet(_ context.Context, key string, values ...interface{}) *redis.IntCmd {
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
	m.store[key] = members
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

func (m *MockRedisClient) ZRem(_ context.Context, key string, members ...interface{}) *redis.IntCmd {
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
	if !found {
		return &res
	}
	valMap1 := val.(map[string]interface{})
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

func (*MockRedisClient) ZRangeArgsWithScores(_ context.Context, _ redis.ZRangeArgs) *redis.ZSliceCmd {
	res := redis.ZSliceCmd{}
	res.SetErr(fmt.Errorf("not implemented"))
	return &res
}
