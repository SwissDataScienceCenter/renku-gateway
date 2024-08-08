// Package redisadapters contains functions for interacting with redis
package db

import (
	"context"
	"encoding"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/mitchellh/mapstructure"
	"github.com/redis/go-redis/v9"
)

const (
	accessTokenPrefix   string = "accessToken-"
	refreshTokenPrefix  string = "refreshToken-"
	idTokenPrefix       string = "idToken-"
	indexExpiringTokens string = "indexExpiringTokens"
	sessionPrefix       string = "session-"
	loginSessionPrefix  string = "loginSession-"
	projectTokenPrefix  string = "projectTokens-"
)

// LimitedRedisClient is the limited set of functionality expected from the redis client in this adapter.
// This allows for easy mocking and swapping of the client. The universal redis client interface is way too big.
type LimitedRedisClient interface {
	HSet(ctx context.Context, key string, values ...any) *redis.IntCmd
	ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	ZRem(ctx context.Context, key string, members ...any) *redis.IntCmd
	HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd
	ZRangeArgsWithScores(ctx context.Context, z redis.ZRangeArgs) *redis.ZSliceCmd

	Persist(ctx context.Context, key string) *redis.BoolCmd
	ExpireAt(ctx context.Context, key string, tm time.Time) *redis.BoolCmd
}

// RedisAdapter contains a redis client
type RedisAdapter struct {
	rdb       LimitedRedisClient
	encryptor models.Encryptor
}

// serializeStruct returns a list of alternativing struct fields and values
// from the provided struct.
// Used to easily save a struct as a Hash in redis. It will only deconstruct exported fields.
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

// deserializeToStruct takes a result from a Hash value in Redis and converts it to a struct
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

// SetSession writes the associated ID, type, expiration and tokenID of a session to Redis
func (r RedisAdapter) SetSession(ctx context.Context, session models.Session) error {
	return r.rdb.HSet(
		ctx,
		sessionPrefix+session.ID,
		r.serializeStruct(session)...,
	).Err()
}

func (r RedisAdapter) setOauthToken(ctx context.Context, token models.AuthToken) error {
	err := validateTokenType(token.Type)
	if err != nil {
		return err
	}
	var keyPrefix string = prefixForTokenType(token.Type)

	if token.Type == models.AccessTokenType {
		if err := r.setToIndexExpiringTokens(ctx, token); err != nil {
			return err
		}
	}

	// if r.encryptor != nil {
	// 	token = token.SetEncryptor(r.encryptor)
	// }
	encToken, err := token.Encrypt(r.encryptor)
	if err != nil {
		return err
	}

	return r.rdb.HSet(
		ctx,
		keyPrefix+token.ID,
		r.serializeStruct(encToken)...,
	).Err()
}

// SetAccessToken writes the associated ID, access token value, expiration, tokenID and refresh URL
// of an access token to Redis.
func (r RedisAdapter) SetAccessToken(ctx context.Context, accessToken models.AuthToken) error {
	if accessToken.Type != models.AccessTokenType {
		return fmt.Errorf("token is not of the right type")
	}
	return r.setOauthToken(ctx, accessToken)
}

// SetRefreshToken writes the associated ID, access token value, expiration and tokenID of a refresh token to Redis
func (r RedisAdapter) SetRefreshToken(ctx context.Context, refreshToken models.AuthToken) error {
	if refreshToken.Type != models.RefreshTokenType {
		return fmt.Errorf("token is not of the right type")
	}
	return r.setOauthToken(ctx, refreshToken)
}

func (r RedisAdapter) SetIDToken(ctx context.Context, idToken models.AuthToken) error {
	if idToken.Type != models.IDTokenType {
		return fmt.Errorf("token is not of the right type")
	}
	return r.setOauthToken(ctx, idToken)
}

// SetToIndexExpiringTokens writes the associated expiration and tokenID of an access token to Redis
func (r RedisAdapter) setToIndexExpiringTokens(ctx context.Context, token models.AuthToken) error {
	var z1 redis.Z
	z1.Score = float64(token.ExpiresAt.Unix())
	z1.Member = prefixForTokenType(token.Type) + token.ID

	return r.rdb.ZAdd(
		ctx,
		indexExpiringTokens,
		z1,
	).Err()
}

// SetProjectToken writes the project ID and associated expiration and tokenID of a project to Redis
func (r RedisAdapter) SetProjectToken(ctx context.Context, projectID int, accessToken models.AuthToken) error {
	z1 := redis.Z{
		Score:  float64(accessToken.ExpiresAt.Unix()),
		Member: accessToken.ID,
	}

	return r.rdb.ZAdd(
		ctx,
		projectTokenPrefix+strconv.Itoa(projectID),
		z1,
	).Err()
}

// Remove/delete functions

// RemoveSession removes a session entry from Redis
func (r RedisAdapter) RemoveSession(ctx context.Context, sessionID string) error {
	return r.rdb.Del(
		ctx,
		sessionPrefix+sessionID,
	).Err()
}

// RemoveAccessToken removes an access token entry from Redis
func (r RedisAdapter) RemoveAccessToken(ctx context.Context, accessToken models.AuthToken) error {
	err := r.removeFromIndexExpiringTokens(ctx, accessToken)
	if err != nil {
		return err
	}

	return r.rdb.Del(
		ctx,
		accessTokenPrefix+accessToken.ID,
	).Err()
}

// RemoveRefreshToken removes an access token entry from Redis
func (r RedisAdapter) RemoveRefreshToken(ctx context.Context, refreshTokenID string) error {
	return r.rdb.Del(
		ctx,
		refreshTokenPrefix+refreshTokenID,
	).Err()
}

func (r RedisAdapter) RemoveIDToken(ctx context.Context, idToken models.AuthToken) error {
	err := r.removeFromIndexExpiringTokens(ctx, idToken)
	if err != nil {
		return err
	}

	return r.rdb.Del(
		ctx,
		idTokenPrefix+idToken.ID,
	).Err()
}

// removeFromIndexExpiringTokens removes a token entry in the indexExpiringTokens sorted set from Redis
func (r RedisAdapter) removeFromIndexExpiringTokens(ctx context.Context, token models.AuthToken) error {
	var z1 redis.Z
	z1.Score = float64(token.ExpiresAt.Unix())
	z1.Member = prefixForTokenType(token.Type) + token.ID

	return r.rdb.ZRem(
		ctx,
		indexExpiringTokens,
		z1,
	).Err()
}

// RemoveProjectToken removes an access token entry in a projectTokens sorted set from Redis
func (r RedisAdapter) RemoveProjectToken(ctx context.Context, projectID int, accessToken models.AuthToken) error {
	var z1 redis.Z
	z1.Score = float64(accessToken.ExpiresAt.Unix())
	z1.Member = accessToken.ID

	return r.rdb.ZRem(
		ctx,
		projectTokenPrefix+strconv.Itoa(projectID),
		z1,
	).Err()
}

// Get functions

// GetSession reads the associated ID, type, expiration and tokenID of a session from Redis
// If the session does not exist in redis then an error is return with value redis.Nil
func (r RedisAdapter) GetSession(ctx context.Context, sessionID string) (models.Session, error) {
	output := models.Session{}
	// NOTE: HGETALL will return an empty list of hash-keys and hash-values if the key is not found
	// then this is deserialized as an empty (zero-valued) struct
	raw, err := r.rdb.HGetAll(
		ctx,
		sessionPrefix+sessionID,
	).Result()
	if err != nil {
		return output, err
	}
	err = r.deserializeToStruct(raw, &output)
	if err != nil {
		if err == gwerrors.ErrMissingDBResource {
			err = gwerrors.ErrSessionNotFound
		}
		return models.Session{}, err
	}
	output.SetTokenStore(r)
	output.SetSessionStore(r)
	return output, nil
}

// getOauthToken reads a specific token from redis, decrypting if necessary.
func (r RedisAdapter) getOauthToken(ctx context.Context, keyPrefix string, tokenID string) (models.AuthToken, error) {
	output := models.AuthToken{}
	raw, err := r.rdb.HGetAll(
		ctx,
		keyPrefix+tokenID,
	).Result()
	if err != nil {
		return output, err
	}

	err = r.deserializeToStruct(raw, &output)
	if err != nil {
		if err == gwerrors.ErrMissingDBResource {
			err = gwerrors.ErrTokenNotFound
		}
		return models.AuthToken{}, err
	}

	// if r.encryptor != nil {
	// 	output = output.SetEncryptor(r.encryptor)
	// }
	decToken, err := output.Decrypt(r.encryptor)
	if err != nil {
		return models.AuthToken{}, err
	}
	return decToken, nil
}

// getOauthTokens is used to query redis (in parallel) for several different tokens at once.
func (r RedisAdapter) getOauthTokens(
	ctx context.Context,
	keyPrefix string,
	tokenIDs ...string,
) (map[string]models.AuthToken, error) {
	if len(tokenIDs) == 0 {
		return map[string]models.AuthToken{}, nil
	}
	wg := sync.WaitGroup{}
	lock := sync.Mutex{}
	wg.Add(len(tokenIDs))
	var tokenErr error
	tokens := make(map[string]models.AuthToken, len(tokenIDs))
	for _, tokenID := range tokenIDs {
		go func(tokenID string) {
			defer wg.Done()
			if tokenErr != nil {
				// NOTE: This means a previous attempt to retrieve a token resulted in an unexpected error
				// so we do not try to get more tokens in this case
				return
			}
			token, err := r.getOauthToken(ctx, keyPrefix, tokenID)
			if err != nil {
				if err == gwerrors.ErrTokenNotFound {
					// Ignore if the token is missing
					return
				}
				tokenErr = err
				return
			}
			lock.Lock()
			defer lock.Unlock()
			tokens[token.ProviderID] = token
			// if err != nil {
			// 	tokenErr = err
			// }
		}(tokenID)
	}
	wg.Wait()
	if tokenErr != nil {
		return map[string]models.AuthToken{}, tokenErr
	}
	return tokens, nil
}

// GetAccessToken reads the associated ID, access token value, expiration, tokenID and refresh URL
// of an access token from Redis
func (r RedisAdapter) GetAccessToken(ctx context.Context, tokenID string) (models.AuthToken, error) {
	return r.getOauthToken(ctx, accessTokenPrefix, tokenID)
}

// GetAccessTokens reads the associated IDs, and returns the tokens in a map keyed by the provider IDs
func (r RedisAdapter) GetAccessTokens(ctx context.Context, tokenIDs ...string) (map[string]models.AuthToken, error) {
	return r.getOauthTokens(ctx, accessTokenPrefix, tokenIDs...)
}

// GetRefreshToken reads the associated ID, refresh token value, expiration and tokenID of a refresh token from Redis
func (r RedisAdapter) GetRefreshToken(ctx context.Context, tokenID string) (models.AuthToken, error) {
	return r.getOauthToken(ctx, refreshTokenPrefix, tokenID)
}

func (r RedisAdapter) GetIDToken(ctx context.Context, tokenID string) (models.AuthToken, error) {
	return r.getOauthToken(ctx, idTokenPrefix, tokenID)
}

func (r RedisAdapter) GetIDTokens(ctx context.Context, tokenIDs ...string) (map[string]models.AuthToken, error) {
	return r.getOauthTokens(ctx, idTokenPrefix, tokenIDs...)
}

// GetRefreshTokens reads the associated IDs, and returns the tokens in map keyed by the provider IDs
func (r RedisAdapter) GetRefreshTokens(ctx context.Context, tokenIDs ...string) (map[string]models.AuthToken, error) {
	return r.getOauthTokens(ctx, refreshTokenPrefix, tokenIDs...)
}

// GetExpiringAccessTokenIDs reads the associated expiration and tokenID of an access token from Redis
func (r RedisAdapter) GetExpiringAccessTokenIDs(
	ctx context.Context,
	startTime time.Time,
	stopTime time.Time,
) ([]string, error) {
	var expiringTokens []string

	zrangeargs := redis.ZRangeArgs{
		Key:     indexExpiringTokens,
		Start:   float64(startTime.Unix()),
		Stop:    float64(stopTime.Unix()),
		ByScore: true,
	}

	zrange, err := r.rdb.ZRangeArgsWithScores(
		ctx,
		zrangeargs,
	).Result()

	for _, expiringToken := range zrange {
		expiringTokens = append(expiringTokens, fmt.Sprintf("%v", expiringToken.Member))
	}

	return expiringTokens, err
}

// GetProjectTokens reads the project ID and associated expiration and tokenID of a project from Redis
func (r RedisAdapter) GetProjectTokens(ctx context.Context, projectID int) ([]string, error) {
	var projectTokens []string

	zrangeargs := redis.ZRangeArgs{
		Key:     projectTokenPrefix + strconv.Itoa(projectID),
		Start:   0,
		Stop:    999999,
		ByScore: false,
	}

	zrange, err := r.rdb.ZRangeArgsWithScores(
		ctx,
		zrangeargs,
	).Result()

	for _, projectToken := range zrange {
		projectTokens = append(projectTokens, fmt.Sprintf("%v", projectToken.Member))
	}

	return projectTokens, err
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

// NewRedisAdapter creates a new DB adapter for Redis, if not provided as an option by default
// it will not use encryption and it will use an in-memory mock of Redis.
func NewRedisAdapter(options ...RedisAdapterOption) (RedisAdapter, error) {
	rdb := RedisAdapter{rdb: &MockRedisClient{}}
	for _, opt := range options {
		err := opt(&rdb)
		if err != nil {
			return RedisAdapter{}, err
		}
	}
	return rdb, nil
}

func prefixForTokenType(tokenType models.OauthTokenType) string {
	switch tokenType {
	case models.AccessTokenType:
		return accessTokenPrefix
	case models.RefreshTokenType:
		return refreshTokenPrefix
	case models.IDTokenType:
		return idTokenPrefix
	default:
		return ""
	}
}

func validateTokenType(tokenType models.OauthTokenType) error {
	switch tokenType {
	case models.AccessTokenType:
		return nil
	case models.RefreshTokenType:
		return nil
	case models.IDTokenType:
		return nil
	default:
		return fmt.Errorf("unknown token type: %s", tokenType)
	}
}
