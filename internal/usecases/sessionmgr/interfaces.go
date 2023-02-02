package sessionmgr

import (
	"context"

	"github.com/SwissDataScienceCenter/renku-gateway-v2/internal/models"
)

type SessionWriter interface {
	WriteSession(context.Context, models.Session) error
}

type SessionRemover interface {
	RemoveSession(ctx context.Context, sessionID string) error
}

type SessionReader interface {
	Read(sessionID string) (models.Session, error)
}

type SessionReaderWriterRemover interface {
	SessionReader
	SessionWriter
	SessionRemover
}
