package models

import (
	"context"
)

type SessionRepository interface {
	SessionGetter
	SessionSetter
	SessionRemover
}

type SessionGetter interface {
	GetSession(ctx context.Context, sessionID string) (Session, error)
}

type SessionSetter interface {
	SetSession(ctx context.Context, session Session) error
}

type SessionRemover interface {
	RemoveSession(ctx context.Context, sessionID string) error
}
