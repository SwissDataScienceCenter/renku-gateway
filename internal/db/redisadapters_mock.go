package db

import (
	"context"
	"encoding"
	"fmt"
	"log"
	"math"
	"slices"
	"strconv"
	"strings"
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

func (m *MockRedisClient) ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	res := redis.IntCmd{}
	// Create a new slice at "key" if there is none
	var zset []any
	zsetRaw, found := m.store[key]
	if found {
		zset_, ok := zsetRaw.([]any)
		if !ok {
			res.SetErr(fmt.Errorf("type error"))
			return &res
		}
		zset = zset_
	} else {
		zset = []any{}
	}
	var newMembers int64 = 0
	for _, z := range members {
		idx := slices.IndexFunc(zset, func(otherZ any) bool {
			asZ, ok := otherZ.(redis.Z)
			if !ok {
				return false
			}
			return z.Member == asZ.Member
		})
		if idx < 0 {
			newMembers = newMembers + 1
			zset = append(zset, z)
		} else {
			zset[idx] = z
		}
	}
	slices.SortFunc(zset, func(a, b any) int {
		aa, aOk := a.(redis.Z)
		bb, bOk := b.(redis.Z)
		if !aOk || !bOk {
			return 0
		}
		return int(aa.Score - bb.Score)
	})
	m.store[key] = zset
	res.SetVal(newMembers)
	return &res
}

func (m *MockRedisClient) ZRangeArgs(ctx context.Context, z redis.ZRangeArgs) *redis.StringSliceCmd {
	res := redis.StringSliceCmd{}

	if z.ByLex {
		res.SetErr(fmt.Errorf("ByLex support not implemented"))
		return &res
	} else if z.Offset != 0 || z.Count != 0 {
		res.SetErr(fmt.Errorf("Offset and Count support not implemented"))
		return &res
	}

	// By index
	if !z.ByScore {
		res.SetErr(fmt.Errorf("ZRange by index support not implemented"))
		return &res
	}

	lower, includeLower, err := parseZStartStop(z.Start)
	if err != nil {
		res.SetErr(fmt.Errorf("could not parse Start: %w", err))
		return &res
	}
	upper, includeUpper, err := parseZStartStop(z.Stop)
	if err != nil {
		res.SetErr(fmt.Errorf("could not parse Stop: %w", err))
		return &res
	}

	zsetRaw, found := m.store[z.Key]
	if !found {
		res.SetVal([]string{})
		return &res
	}
	zset, ok := zsetRaw.([]any)
	if !ok {
		res.SetErr(fmt.Errorf("type error"))
		return &res
	}

	slice := []string{}
	for _, zz := range zset {
		asZ, ok := zz.(redis.Z)
		if !ok {
			res.SetErr(fmt.Errorf("member type error"))
			return &res
		}
		if (!includeUpper && asZ.Score >= upper) || asZ.Score > upper {
			break
		}
		if (includeLower && asZ.Score >= lower) || asZ.Score > lower {
			slice = append(slice, fmt.Sprintf("%v", asZ.Member))
		}
	}

	res.SetVal(slice)
	return &res
}

func parseZStartStop(x any) (value float64, include bool, err error) {
	value = math.Inf(-1)
	include = true

	valueStr, ok := x.(string)
	if !ok {
		valueStr = fmt.Sprintf("%v", x)
	}

	if strings.HasPrefix(valueStr, "(") {
		include = false
		value_, err := strconv.ParseFloat(strings.TrimPrefix(valueStr, "("), 64)
		if err != nil {
			return value, include, err
		}
		value = value_
	} else {
		value_, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			return value, include, err
		}
		value = value_
	}

	return value, include, nil
}
