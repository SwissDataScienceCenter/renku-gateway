package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

const (
	accessTokenPrefix  string = "accessToken"
	refreshTokenPrefix string = "refreshToken"
	idTokenPrefix      string = "idToken"
)

const tokenExpiresAtLeeway time.Duration = 10 * time.Second

// GetAccessToken reads the associated ID, access token value, expiration, tokenID and refresh URL
// of an access token from Redis
func (r RedisAdapter) GetAccessToken(ctx context.Context, tokenID string) (models.AuthToken, error) {
	return r.getAuthToken(ctx, r.accessTokenKey(tokenID))
}

// GetRefreshToken reads the associated ID, refresh token value, expiration and tokenID of a refresh token from Redis
func (r RedisAdapter) GetRefreshToken(ctx context.Context, tokenID string) (models.AuthToken, error) {
	return r.getAuthToken(ctx, r.refreshTokenKey(tokenID))
}

func (r RedisAdapter) GetIDToken(ctx context.Context, tokenID string) (models.AuthToken, error) {
	return r.getAuthToken(ctx, r.idTokenKey(tokenID))
}

// SetAccessToken writes the associated ID, access token value, expiration, tokenID and refresh URL
// of an access token to Redis.
func (r RedisAdapter) SetAccessToken(ctx context.Context, token models.AuthToken) error {
	if token.Type != models.AccessTokenType {
		return fmt.Errorf("token is not of the right type")
	}
	return r.setAuthToken(ctx, token)
}

func (r RedisAdapter) SetAccessTokenExpiry(ctx context.Context, token models.AuthToken, expiresAt time.Time) error {
	if token.Type != models.AccessTokenType {
		return fmt.Errorf("token is not of the right type")
	}
	return r.setAuthTokenExpiry(ctx, token, expiresAt)
}

// SetRefreshToken writes the associated ID, access token value, expiration and tokenID of a refresh token to Redis
func (r RedisAdapter) SetRefreshToken(ctx context.Context, token models.AuthToken) error {
	if token.Type != models.RefreshTokenType {
		return fmt.Errorf("token is not of the right type")
	}
	return r.setAuthToken(ctx, token)
}

func (r RedisAdapter) SetRefreshTokenExpiry(ctx context.Context, token models.AuthToken, expiresAt time.Time) error {
	if token.Type != models.RefreshTokenType {
		return fmt.Errorf("token is not of the right type")
	}
	return r.setAuthTokenExpiry(ctx, token, expiresAt)
}

func (r RedisAdapter) SetIDToken(ctx context.Context, token models.AuthToken) error {
	if token.Type != models.IDTokenType {
		return fmt.Errorf("token is not of the right type")
	}
	return r.setAuthToken(ctx, token)
}

func (r RedisAdapter) SetIDTokenExpiry(ctx context.Context, token models.AuthToken, expiresAt time.Time) error {
	if token.Type != models.IDTokenType {
		return fmt.Errorf("token is not of the right type")
	}
	return r.setAuthTokenExpiry(ctx, token, expiresAt)
}

func (RedisAdapter) accessTokenKey(tokenID string) string {
	return accessTokenPrefix + ":" + tokenID
}

func (RedisAdapter) refreshTokenKey(tokenID string) string {
	return refreshTokenPrefix + ":" + tokenID
}

func (RedisAdapter) idTokenKey(tokenID string) string {
	return idTokenPrefix + ":" + tokenID
}

func (r RedisAdapter) getTokenKey(token models.AuthToken) string {
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
func (r RedisAdapter) getAuthToken(ctx context.Context, key string) (models.AuthToken, error) {
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

func (r RedisAdapter) setAuthToken(ctx context.Context, token models.AuthToken) error {
	err := validateTokenType(token.Type)
	if err != nil {
		return err
	}

	encToken, err := token.Encrypt(r.encryptor)
	if err != nil {
		return err
	}

	slog.Debug(
		"TOKEN STORE",
		"message",
		"saving token",
		"token",
		token,
	)

	return r.rdb.HSet(
		ctx,
		r.getTokenKey(token),
		r.serializeStruct(encToken)...,
	).Err()
}

func (r RedisAdapter) setAuthTokenExpiry(ctx context.Context, token models.AuthToken, expiresAt time.Time) error {
	err := validateTokenType(token.Type)
	if err != nil {
		return err
	}

	key := r.getTokenKey(token)
	if expiresAt.IsZero() {
		return r.rdb.Persist(ctx, key).Err()
	}
	return r.rdb.ExpireAt(ctx, key, expiresAt.Add(tokenExpiresAtLeeway)).Err()
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
