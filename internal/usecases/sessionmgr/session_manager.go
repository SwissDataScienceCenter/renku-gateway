package sessionmgr

import (
	"context"

	"github.com/SwissDataScienceCenter/renku-gateway-v2/internal/models"
)

type SessionManager struct {
	Store SessionWriter
}

func (s *SessionManager) Refresh(session models.Session) (newSession models.Session, err error) {
	s.Store.WriteSession(context.Background(), session)
	return models.Session{}, nil
}

func (s *SessionManager) Logout(sessionID string) (err error) {
	//s.store.Remove(sessionID)
	return nil
}
