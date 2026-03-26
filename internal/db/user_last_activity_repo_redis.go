package db

import (
	"context"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

const (
	userLastActivityPrefix string = "userLastActivity"
)

func (r RedisAdapter) GetUserLastActivity(ctx context.Context, userID string) (models.UserLastActivity, error) {
	output := models.UserLastActivity{}
	// NOTE: HGETALL will return an empty list of hash-keys and hash-values if the key is not found
	// then this is deserialized as an empty (zero-valued) struct
	raw, err := r.rdb.HGetAll(
		ctx,
		userLastActivityKey(userID),
	).Result()
	if err != nil {
		return output, err
	}
	err = r.deserializeToStruct(raw, &output)
	if err != nil {
		if err == gwerrors.ErrMissingDBResource {
			err = gwerrors.ErrNotFound
		}
		return models.UserLastActivity{}, err
	}
	return output, nil
}

func (r RedisAdapter) SetUserLastActivity(ctx context.Context, userLastActivity models.UserLastActivity) error {
	key := userLastActivityKey(userLastActivity.UserID)
	err := r.rdb.HSet(
		ctx,
		key,
		r.serializeStruct(userLastActivity)...,
	).Err()
	if err != nil {
		return err
	}
	return r.rdb.ExpireAt(ctx, key, userLastActivity.ExpiresAt.Add(tokenExpiresAtLeeway)).Err()
}

func userLastActivityKey(userID string) string {
	return userLastActivityPrefix + ":" + userID
}
