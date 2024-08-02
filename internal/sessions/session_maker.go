package sessions

import (
	"log/slog"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

type SessionMaker interface {
	NewSession() (Session, error)
}

type SessionMakerImpl struct {
	idleSessionTTLSeconds int
	maxSessionTTLSeconds  int
}

var randomIDGenerator models.IDGenerator = models.RandomGenerator{Length: 24}

func (sm *SessionMakerImpl) NewSession() (Session, error) {
	id, err := randomIDGenerator.ID()
	if err != nil {
		return Session{}, err
	}
	session := Session{
		ID:             id,
		CreatedAt:      time.Now().UTC(),
		IdleTTLSeconds: models.SerializableInt(sm.idleSessionTTLSeconds),
		MaxTTLSeconds:  models.SerializableInt(sm.maxSessionTTLSeconds),
	}
	session.ExpiresAt = session.CreatedAt.Add(session.IdleTTL())
	slog.Info("NEW SESSION", "session", session)
	return session, nil
}

type SessionMakerOption func(*SessionMakerImpl) error

func WithIdleSessionTTLSeconds(s int) SessionMakerOption {
	return func(sm *SessionMakerImpl) error {
		sm.idleSessionTTLSeconds = s
		return nil
	}
}

func WithMaxSessionTTLSeconds(s int) SessionMakerOption {
	return func(sm *SessionMakerImpl) error {
		sm.maxSessionTTLSeconds = s
		return nil
	}
}

func NewSessionMaker(options ...SessionMakerOption) SessionMaker {
	sm := SessionMakerImpl{idleSessionTTLSeconds: 0, maxSessionTTLSeconds: 0}
	for _, opt := range options {
		opt(&sm)
	}
	return &sm
}
