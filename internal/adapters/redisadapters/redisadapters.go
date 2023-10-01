// Package redisadapters contains functions for interacting with redis
package redisadapters

import (
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway-v2/internal/models"
	"github.com/go-redis/redis/v9"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/net/context"
)

const (
	accessTokenPrefix   string = "accessToken-"
	refreshTokenPrefix  string = "refreshToken-"
	indexExpiringTokens string = "indexExpiringTokens"
	sessionPrefix       string = "session-"
	loginSessionPrefix  string = "loginSession-"
	projectTokenPrefix  string = "projectTokens-"
)

// LimitedRedisClient is the limited set of functionality expected from the redis client in this adapter.
// This allows for easy mocking and swapping of the client. The universal redis client interface is way too big.
type LimitedRedisClient interface {
	HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd
	ZRangeArgsWithScores(ctx context.Context, z redis.ZRangeArgs) *redis.ZSliceCmd
}

// RedisAdapter contains a redis client
type RedisAdapter struct {
	rdb       LimitedRedisClient
	encryptor models.Encryptor
}

// serializeStruct returns a list of alternativing struct fields and values
// from the provided struct.
// Used to easily save a struct as a Hash in redis. It will only deconstruct exported fields.
func (RedisAdapter) serializeStruct(strct interface{}) []interface{} {
	v := reflect.ValueOf(strct)
	t := v.Type()
	var output []interface{}
	for i := 0; i < v.NumField(); i++ {
		if !t.Field(i).IsExported() {
			continue
		}
		fieldName := t.Field(i).Name
		fieldValue := v.Field(i).Interface()
		output = append(output, fieldName, fieldValue)
	}
	return output
}

// deserializeToStruct takes a result from a Hash value in Redis and converts it to a struct
func (RedisAdapter) deserializeToStruct(hash map[string]string, output interface{}) error {
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

// Set/write functions

// SetSession writes the associated ID, type, expiration and tokenID of a session to Redis
func (r RedisAdapter) SetSession(ctx context.Context, session models.Session) error {
	return r.rdb.HSet(
		ctx,
		sessionPrefix+session.ID,
		r.serializeStruct(session)...,
	).Err()
}

func (r RedisAdapter) setOauthToken(ctx context.Context, token models.OauthToken) error {
	var keyPrefix string
	switch token.Type {
	case models.AccessTokenType:
		keyPrefix = accessTokenPrefix
	case models.RefreshTokenType:
		keyPrefix = refreshTokenPrefix
	default:
		return fmt.Errorf("unknown token type")
	}

	if token.Type == models.AccessTokenType {
		if err := r.setToIndexExpiringTokens(ctx, token); err != nil {
			return err
		}
	}

	encToken, err := token.Encrypt()
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
func (r RedisAdapter) SetAccessToken(ctx context.Context, accessToken models.OauthToken) error {
	if accessToken.Type != models.AccessTokenType {
		return fmt.Errorf("token is not of the right type")
	}
	return r.setOauthToken(ctx, accessToken)
}

// SetRefreshToken writes the associated ID, access token value, expiration and tokenID of a refresh token to Redis
func (r RedisAdapter) SetRefreshToken(ctx context.Context, refreshToken models.OauthToken) error {
	if refreshToken.Type != models.RefreshTokenType {
		return fmt.Errorf("token is not of the right type")
	}
	return r.setOauthToken(ctx, refreshToken)
}

// SetToIndexExpiringTokens writes the associated expiration and tokenID of an access token to Redis
func (r RedisAdapter) setToIndexExpiringTokens(ctx context.Context, accessToken models.OauthToken) error {
	var z1 redis.Z
	z1.Score = float64(accessToken.ExpiresAt.Unix())
	z1.Member = accessToken.ID

	return r.rdb.ZAdd(
		ctx,
		indexExpiringTokens,
		z1,
	).Err()
}

// SetProjectToken writes the project ID and associated expiration and tokenID of a project to Redis
func (r RedisAdapter) SetProjectToken(ctx context.Context, projectID int, accessToken models.OauthToken) error {
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
func (r RedisAdapter) RemoveAccessToken(ctx context.Context, accessToken models.OauthToken) error {
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

// removeFromIndexExpiringTokens removes an access token entry in the indexExpiringTokens sorted set from Redis
func (r RedisAdapter) removeFromIndexExpiringTokens(ctx context.Context, accessToken models.OauthToken) error {
	var z1 redis.Z
	z1.Score = float64(accessToken.ExpiresAt.Unix())
	z1.Member = accessToken.ID

	return r.rdb.ZRem(
		ctx,
		indexExpiringTokens,
		z1,
	).Err()
}

// RemoveProjectToken removes an access token entry in a projectTokens sorted set from Redis
func (r RedisAdapter) RemoveProjectToken(ctx context.Context, projectID int, accessToken models.OauthToken) error {
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
func (r RedisAdapter) GetSession(ctx context.Context, sessionID string) (models.Session, error) {
	output := models.Session{}
	raw, err := r.rdb.HGetAll(
		ctx,
		sessionPrefix+sessionID,
	).Result()
	if err != nil {
		return output, err
	}

	err = r.deserializeToStruct(raw, &output)
	return output, err
}

// getOauthToken reads a specific token from redis, decrypting if necessary.
func (r RedisAdapter) getOauthToken(ctx context.Context, keyPrefix string, tokenID string) (models.OauthToken, error) {
	output := models.OauthToken{}
	raw, err := r.rdb.HGetAll(
		ctx,
		keyPrefix+tokenID,
	).Result()
	if err != nil {
		return output, err
	}

	err = r.deserializeToStruct(raw, &output)
	if err != nil {
		return models.OauthToken{}, err
	}

	if r.encryptor != nil {
		output = output.SetEncryptor(r.encryptor)
	}
	decToken, err := output.Decrypt()
	if err != nil {
		return models.OauthToken{}, err
	}
	return decToken, nil
}

// getOauthTokens is used to query redis (in parallel) for several different tokens at once.
func (r RedisAdapter) getOauthTokens(
	ctx context.Context,
	keyPrefix string,
	tokenIDs ...string,
) (map[string]models.OauthToken, error) {
	if len(tokenIDs) == 0 {
		return map[string]models.OauthToken{}, nil
	}
	wg := sync.WaitGroup{}
	lock := sync.Mutex{}
	wg.Add(len(tokenIDs))
	var tokenErr error
	tokens := make(map[string]models.OauthToken, len(tokenIDs))
	for _, tokenID := range tokenIDs {
		go func(tokenID string) {
			defer wg.Done()
			token, err := r.getOauthToken(ctx, keyPrefix, tokenID)
			lock.Lock()
			defer lock.Unlock()
			tokens[token.ProviderID] = token
			if err != nil {
				tokenErr = err
			}
		}(tokenID)
	}
	wg.Wait()
	if tokenErr != nil {
		return map[string]models.OauthToken{}, tokenErr
	}
	return tokens, nil
}

// GetAccessToken reads the associated ID, access token value, expiration, tokenID and refresh URL
// of an access token from Redis
func (r RedisAdapter) GetAccessToken(ctx context.Context, tokenID string) (models.OauthToken, error) {
	return r.getOauthToken(ctx, accessTokenPrefix, tokenID)
}

// GetAccessTokens reads the associated IDs, and returns the tokens in a map keyed by the provider IDs
func (r RedisAdapter) GetAccessTokens(ctx context.Context, tokenIDs ...string) (map[string]models.OauthToken, error) {
	return r.getOauthTokens(ctx, accessTokenPrefix, tokenIDs...)
}

// GetRefreshToken reads the associated ID, refresh token value, expiration and tokenID of a refresh token from Redis
func (r RedisAdapter) GetRefreshToken(ctx context.Context, tokenID string) (models.OauthToken, error) {
	return r.getOauthToken(ctx, refreshTokenPrefix, tokenID)
}

// GetRefreshTokens reads the associated IDs, and returns the tokens in map keyed by the provider IDs
func (r RedisAdapter) GetRefreshTokens(ctx context.Context, tokenIDs ...string) (map[string]models.OauthToken, error) {
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
		Start:   startTime.Unix(),
		Stop:    stopTime.Unix(),
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

func NewRedisAdapter(rdb redis.UniversalClient, encryptor models.Encryptor) RedisAdapter {
	return RedisAdapter{rdb: rdb, encryptor: encryptor}
}
