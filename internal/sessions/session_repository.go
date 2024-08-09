package sessions

import (
	"context"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

// TODO: move to models
type SessionRepository interface {
	SessionGetter
	SessionSetter
	SessionRemover
}

type SessionGetter interface {
	GetSession(ctx context.Context, sessionID string) (models.Session, error)
}

type SessionSetter interface {
	SetSession(ctx context.Context, session models.Session) error
}

type SessionRemover interface {
	RemoveSession(ctx context.Context, sessionID string) error
}
