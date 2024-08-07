package dbnew

import (
	"context"
	"encoding"
	"fmt"
	"reflect"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/db"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
	"github.com/mitchellh/mapstructure"
	"github.com/redis/go-redis/v9"
)

const (
	sessionPrefix      string = "session"
	accessTokenPrefix  string = "accessToken"
	refreshTokenPrefix string = "refreshToken"
	idTokenPrefix      string = "idToken"
	expiringTokensKey  string = "expiringTokens"
)

const expiresAtLeeway time.Duration = 10 * time.Second

type RedisAdapterNew struct {
	rdb       db.LimitedRedisClient
	encryptor models.Encryptor
}

func (r RedisAdapterNew) GetSession(ctx context.Context, sessionID string) (sessions.Session, error) {
	output := sessions.Session{}
	// NOTE: HGETALL will return an empty list of hash-keys and hash-values if the key is not found
	// then this is deserialized as an empty (zero-valued) struct
	raw, err := r.rdb.HGetAll(
		ctx,
		r.sessionKey(sessionID),
	).Result()
	if err != nil {
		return output, err
	}
	err = r.deserializeToStruct(raw, &output)
	if err != nil {
		if err == gwerrors.ErrMissingDBResource {
			err = gwerrors.ErrSessionNotFound
		}
		return sessions.Session{}, err
	}
	return output, nil
}

func (r RedisAdapterNew) SetSession(ctx context.Context, session sessions.Session) error {
	key := r.sessionKey(session.ID)
	err := r.rdb.HSet(
		ctx,
		key,
		r.serializeStruct(session)...,
	).Err()
	if err != nil {
		return nil
	}
	return r.rdb.PExpireAt(ctx, key, session.ExpiresAt.Add(expiresAtLeeway)).Err()
}

func (r RedisAdapterNew) RemoveSession(ctx context.Context, sessionID string) error {
	return r.rdb.Del(
		ctx,
		r.sessionKey(sessionID),
	).Err()
}

func (RedisAdapterNew) sessionKey(sessionID string) string {
	return sessionPrefix + ":" + sessionID
}

// GetAccessToken reads the associated ID, access token value, expiration, tokenID and refresh URL
// of an access token from Redis
func (r RedisAdapterNew) GetAccessToken(ctx context.Context, tokenID string) (models.AuthToken, error) {
	return r.getAuthToken(ctx, r.accessTokenKey(tokenID))
}

// GetRefreshToken reads the associated ID, refresh token value, expiration and tokenID of a refresh token from Redis
func (r RedisAdapterNew) GetRefreshToken(ctx context.Context, tokenID string) (models.AuthToken, error) {
	return r.getAuthToken(ctx, r.refreshTokenKey(tokenID))
}

func (r RedisAdapterNew) GetIDToken(ctx context.Context, tokenID string) (models.AuthToken, error) {
	return r.getAuthToken(ctx, r.idTokenKey(tokenID))
}

// SetAccessToken writes the associated ID, access token value, expiration, tokenID and refresh URL
// of an access token to Redis.
func (r RedisAdapterNew) SetAccessToken(ctx context.Context, session sessions.Session, token models.AuthToken) error {
	if token.Type != models.AccessTokenType {
		return fmt.Errorf("token is not of the right type")
	}
	return r.setAuthToken(ctx, session, token)
}

// SetRefreshToken writes the associated ID, access token value, expiration and tokenID of a refresh token to Redis
func (r RedisAdapterNew) SetRefreshToken(ctx context.Context, session sessions.Session, refreshToken models.AuthToken) error {
	if refreshToken.Type != models.RefreshTokenType {
		return fmt.Errorf("token is not of the right type")
	}
	return r.setAuthToken(ctx, session, refreshToken)
}

func (r RedisAdapterNew) SetIDToken(ctx context.Context, session sessions.Session, idToken models.AuthToken) error {
	if idToken.Type != models.IDTokenType {
		return fmt.Errorf("token is not of the right type")
	}
	return r.setAuthToken(ctx, session, idToken)
}

func (RedisAdapterNew) accessTokenKey(tokenID string) string {
	return accessTokenPrefix + ":" + tokenID
}

func (RedisAdapterNew) refreshTokenKey(tokenID string) string {
	return refreshTokenPrefix + ":" + tokenID
}

func (RedisAdapterNew) idTokenKey(tokenID string) string {
	return idTokenPrefix + ":" + tokenID
}

func (r RedisAdapterNew) getTokenKey(token models.AuthToken) string {
	switch token.Type {
	case models.AccessTokenType:
		return r.accessTokenKey(token.ID)
	case models.RefreshTokenType:
		return r.refreshTokenKey(token.ID)
	case models.IDTokenType:
		return r.idTokenKey(token.ID)
	default:
		return "unknown:" + token.ID
	}
}

// getAuthToken reads a specific token from redis, decrypting if necessary.
func (r RedisAdapterNew) getAuthToken(ctx context.Context, key string) (models.AuthToken, error) {
	output := models.AuthToken{}
	raw, err := r.rdb.HGetAll(
		ctx,
		key,
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

	decToken, err := output.Decrypt(r.encryptor)
	if err != nil {
		return models.AuthToken{}, err
	}
	return decToken, nil
}
func (r RedisAdapterNew) setAuthToken(ctx context.Context, session sessions.Session, token models.AuthToken) error {
	err := validateTokenType(token.Type)
	if err != nil {
		return err
	}

	if token.Type == models.AccessTokenType {
		if err := r.setAccessTokenExpiry(ctx, token); err != nil {
			return err
		}
	}

	encToken, err := token.Encrypt(r.encryptor)
	if err != nil {
		return err
	}

	key := r.getTokenKey(token)
	err = r.rdb.HSet(
		ctx,
		key,
		r.serializeStruct(encToken)...,
	).Err()
	if err != nil {
		return nil
	}
	return r.rdb.PExpireAt(ctx, key, session.CreatedAt.Add(session.MaxTTL()).Add(expiresAtLeeway)).Err()
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

func (r RedisAdapterNew) setAccessTokenExpiry(ctx context.Context, token models.AuthToken) error {
	z := redis.Z{
		Score:  float64(token.ExpiresAt.Unix()),
		Member: r.accessTokenKey(token.ID),
	}
	return r.rdb.ZAdd(ctx, expiringTokensKey, z).Err()
}

func (RedisAdapterNew) serializeStruct(strct any) []any {
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

func (RedisAdapterNew) deserializeToStruct(hash map[string]string, output any) error {
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

// GetExpiringAccessTokenIDs reads the associated expiration and tokenID of an access token from Redis
func (r RedisAdapterNew) GetExpiringAccessTokenIDs(ctx context.Context, expiryEnd time.Time) ([]string, error) {
	var expiringTokens []string

	zrangeargs := redis.ZRangeArgs{
		Key:     expiringTokensKey,
		Start:   "-inf",
		Stop:    float64(expiryEnd.Unix()),
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

type RedisAdapterNewOption func(*RedisAdapterNew) error

func WithRedisConfig(redisConfig config.RedisConfig) RedisAdapterNewOption {
	return func(r *RedisAdapterNew) error {
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
		// case config.DBTypeRedisMock:
		// 	r.rdb = &MockRedisClient{map[string]any{}}
		// 	return nil
		default:
			return fmt.Errorf("unrecognized persistence type %v", redisConfig.Type)
		}
	}
}

func WithEcryption(secretKey string) RedisAdapterNewOption {
	return func(r *RedisAdapterNew) error {
		encryptor, err := db.NewGCMEncryptor(secretKey)
		if err != nil {
			return err
		}
		r.encryptor = encryptor
		return nil
	}
}

func NewRedisAdapterNew(options ...RedisAdapterNewOption) (RedisAdapterNew, error) {
	db := RedisAdapterNew{}
	for _, opt := range options {
		err := opt(&db)
		if err != nil {
			return RedisAdapterNew{}, err
		}
	}
	if db.rdb == nil {
		return RedisAdapterNew{}, fmt.Errorf("redis client is not initialized")
	}
	return db, nil
}
