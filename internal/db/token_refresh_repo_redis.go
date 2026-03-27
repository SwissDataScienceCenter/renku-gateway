package db

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const refreshTokenExpirySortedSetKey string = "refreshTokensExpiry"

// GetExpiringRefreshTokenIDs returns the token IDs of refresh tokens which will expire in the time range [from, to]
func (r RedisAdapter) GetExpiringRefreshTokenIDs(ctx context.Context, from, to time.Time) ([]string, error) {
	start := float64(from.Add(-tokenExpiresAtLeeway).Unix())
	stop := float64(to.Add(tokenExpiresAtLeeway).Unix())
	return r.rdb.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     refreshTokenExpirySortedSetKey,
		ByScore: true,
		Start:   start,
		Stop:    stop,
	}).Result()
}
