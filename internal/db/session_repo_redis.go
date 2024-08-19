package db

import (
	"context"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

const (
	sessionPrefix string = "session"
)

func (r RedisAdapter) GetSession(ctx context.Context, sessionID string) (models.Session, error) {
	output := models.Session{}
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
		return models.Session{}, err
	}
	return output, nil
}

func (r RedisAdapter) SetSession(ctx context.Context, session models.Session) error {
	key := r.sessionKey(session.ID)
	err := r.rdb.HSet(
		ctx,
		key,
		r.serializeStruct(session)...,
	).Err()
	if err != nil {
		return nil
	}
	return r.rdb.ExpireAt(ctx, key, session.ExpiresAt.Add(tokenExpiresAtLeeway)).Err()
}

func (r RedisAdapter) RemoveSession(ctx context.Context, sessionID string) error {
	return r.rdb.Del(
		ctx,
		r.sessionKey(sessionID),
	).Err()
}

func (RedisAdapter) sessionKey(sessionID string) string {
	return sessionPrefix + ":" + sessionID
}
