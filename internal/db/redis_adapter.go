package db

import (
	"encoding"
	"fmt"
	"reflect"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/mitchellh/mapstructure"
	"github.com/redis/go-redis/v9"
)

type RedisAdapter struct {
	rdb       LimitedRedisClient
	encryptor models.Encryptor
}

func (RedisAdapter) serializeStruct(strct any) []any {
	v := reflect.ValueOf(strct)
	t := v.Type()
	var output []any
	for i := 0; i < v.NumField(); i++ {
		if !t.Field(i).IsExported() {
			continue
		}
		fieldName := t.Field(i).Name
		fieldValue := v.Field(i).Interface()
		marshaller, ok := fieldValue.(encoding.TextMarshaler)
		if !ok {
			output = append(output, fieldName, fieldValue)
			continue
		}
		rawBytes, err := marshaller.MarshalText()
		if err != nil {
			output = append(output, fieldName, fieldValue)
			continue
		}
		output = append(output, fieldName, string(rawBytes))
	}
	return output
}

func (RedisAdapter) deserializeToStruct(hash map[string]string, output any) error {
	if len(hash) == 0 {
		// HGetAll returns an empty list of keys and values if the element is not present in the DB
		// then this is deserialized the empty valued struct of whatever it is we are looking at
		return gwerrors.ErrMissingDBResource
	}
	decoder, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				mapstructure.TextUnmarshallerHookFunc(),
			),
			Result: output,
		},
	)
	if err != nil {
		return err
	}
	return decoder.Decode(hash)
}

type RedisAdapterOption func(*RedisAdapter) error

func WithRedisConfig(redisConfig config.RedisConfig) RedisAdapterOption {
	return func(r *RedisAdapter) error {
		switch redisConfig.Type {
		case config.DBTypeRedis:
			if redisConfig.IsSentinel {
				rdb := redis.NewFailoverClient(&redis.FailoverOptions{
					MasterName:       redisConfig.MasterName,
					SentinelAddrs:    redisConfig.Addresses,
					Password:         string(redisConfig.Password),
					DB:               redisConfig.DBIndex,
					SentinelPassword: string(redisConfig.Password),
				})
				r.rdb = rdb
				return nil
			}
			rdb := redis.NewClient(&redis.Options{
				Password: string(redisConfig.Password),
				DB:       redisConfig.DBIndex,
				Addr:     redisConfig.Addresses[0],
			})
			r.rdb = rdb
			return nil
		case config.DBTypeRedisMock:
			r.rdb = &MockRedisClient{map[string]any{}}
			return nil
		default:
			return fmt.Errorf("unrecognized persistence type %v", redisConfig.Type)
		}
	}
}

func WithEcryption(secretKey string) RedisAdapterOption {
	return func(r *RedisAdapter) error {
		encryptor, err := NewGCMEncryptor(secretKey)
		if err != nil {
			return err
		}
		r.encryptor = encryptor
		return nil
	}
}

func NewRedisAdapter(options ...RedisAdapterOption) (*RedisAdapter, error) {
	db := RedisAdapter{}
	for _, opt := range options {
		err := opt(&db)
		if err != nil {
			return &RedisAdapter{}, err
		}
	}
	if db.rdb == nil {
		return &RedisAdapter{}, fmt.Errorf("redis client is not initialized")
	}
	return &db, nil
}
