package sessions

import (
	"context"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

type TokenRefresher interface {
	GetFreshAccessToken(ctx context.Context, tokenID string) (models.AuthToken, error)
}
