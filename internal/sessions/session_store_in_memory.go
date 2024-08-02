package sessions

import (
	"context"
	"sync"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
)

type InMemorySessionStore struct {
	lock     *sync.RWMutex
	sessions map[string]Session
}

func (db *InMemorySessionStore) GetSession(ctx context.Context, id string) (Session, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()
	session, found := db.sessions[id]
	if !found {
		return Session{}, gwerrors.ErrSessionNotFound
	}
	return session, nil
}

func (db *InMemorySessionStore) SetSession(ctx context.Context, session Session) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	db.sessions[session.ID] = session
	return nil
}

func (db *InMemorySessionStore) RemoveSession(ctx context.Context, id string) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	session, found := db.sessions[id]
	if !found {
		return nil
	}
	delete(db.sessions, session.ID)
	return nil
}

func NewInMemorySessionStore() InMemorySessionStore {
	db := InMemorySessionStore{lock: &sync.RWMutex{}, sessions: map[string]Session{}}
	return db
}
