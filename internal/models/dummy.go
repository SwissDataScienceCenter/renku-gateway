package models

import (
	"context"
	"sync"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
)

type DummyDBAdapter struct {
	lock          *sync.RWMutex
	accessTokens  map[string]OauthToken
	refreshTokens map[string]OauthToken
	idTokens       map[string]OauthToken
	sessions      map[string]Session
}

type DummyAdapterOption func(*DummyDBAdapter)

func WithAccessTokens(tokens ...OauthToken) DummyAdapterOption {
	return func(d *DummyDBAdapter) {
		for _, token := range tokens {
			d.accessTokens[token.ID] = token
		}
	}
}

func WithRefreshTokens(tokens ...OauthToken) DummyAdapterOption {
	return func(d *DummyDBAdapter) {
		for _, token := range tokens {
			d.refreshTokens[token.ID] = token
		}
	}
}

func WithSessions(sessions ...Session) DummyAdapterOption {
	return func(d *DummyDBAdapter) {
		for _, session := range sessions {
			d.sessions[session.ID] = session
		}
	}
}

func NewDummyDBAdapter(options ...DummyAdapterOption) DummyDBAdapter {
	db := DummyDBAdapter{lock: &sync.RWMutex{}, accessTokens: map[string]OauthToken{}, refreshTokens: map[string]OauthToken{}, idTokens: map[string]OauthToken{}, sessions: map[string]Session{}}
	for _, opt := range options {
		opt(&db)
	}
	return db
}

func (d *DummyDBAdapter) GetSession(ctx context.Context, id string) (Session, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	session, found := d.sessions[id]
	if !found {
		return Session{}, gwerrors.ErrSessionNotFound
	}
	return session, nil
}

func (d *DummyDBAdapter) SetSession(ctx context.Context, session Session) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.sessions[session.ID] = session
	return nil
}

func (d *DummyDBAdapter) RemoveSession(ctx context.Context, id string) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	session, found := d.sessions[id]
	if !found {
		return nil
	}
	delete(d.sessions, session.ID)
	return nil
}

func (d *DummyDBAdapter) GetAccessToken(ctx context.Context, id string) (OauthToken, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	token, found := d.accessTokens[id]
	if !found {
		return OauthToken{}, gwerrors.ErrTokenNotFound
	}
	return token, nil
}

func (d *DummyDBAdapter) GetAccessTokens(ctx context.Context, ids ...string) (map[string]OauthToken, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	tokens := map[string]OauthToken{}
	for _, id := range ids {
		token, found := d.accessTokens[id]
		if found {
			tokens[token.ProviderID] = token
		}
	}
	return tokens, nil
}

func (d *DummyDBAdapter) GetRefreshToken(ctx context.Context, id string) (OauthToken, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	token, found := d.refreshTokens[id]
	if !found {
		return OauthToken{}, gwerrors.ErrTokenNotFound
	}
	return token, nil
}

func (d *DummyDBAdapter) GetRefreshTokens(ctx context.Context, ids ...string) (map[string]OauthToken, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	tokens := map[string]OauthToken{}
	for _, id := range ids {
		token, found := d.refreshTokens[id]
		if found {
			tokens[token.ProviderID] = token
		}
	}
	return tokens, nil
}

func (d *DummyDBAdapter) GetIDToken(ctx context.Context, id string) (OauthToken, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	token, found := d.idTokens[id]
	if !found {
		return OauthToken{}, gwerrors.ErrTokenNotFound
	}
	return token, nil
}

func (d *DummyDBAdapter) GetIDTokens(ctx context.Context, ids ...string) (map[string]OauthToken, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	tokens := map[string]OauthToken{}
	for _, id := range ids {
		token, found := d.idTokens[id]
		if found {
			tokens[token.ProviderID] = token
		}
	}
	return tokens, nil
}

func (d *DummyDBAdapter) GetExpiringAccessTokenIDs(ctx context.Context, start time.Time, end time.Time) ([]string, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	ids := []string{}
	for id, token := range d.accessTokens {
		if token.ExpiresAt.After(start) && token.ExpiresAt.Before(end) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (d *DummyDBAdapter) SetAccessToken(ctx context.Context, token OauthToken) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.accessTokens[token.ID] = token
	return nil
}

func (d *DummyDBAdapter) SetRefreshToken(ctx context.Context, token OauthToken) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.refreshTokens[token.ID] = token
	return nil
}

func (d *DummyDBAdapter) SetIDToken(ctx context.Context, token OauthToken) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.idTokens[token.ID] = token
	return nil
}

func (d *DummyDBAdapter) RemoveAccessToken(ctx context.Context, token OauthToken) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	_, found := d.accessTokens[token.ID]
	if !found {
		return nil
	}
	delete(d.accessTokens, token.ID)
	return nil
}

func (d *DummyDBAdapter) RemoveRefreshToken(ctx context.Context, id string) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	_, found := d.refreshTokens[id]
	if !found {
		return nil
	}
	delete(d.refreshTokens, id)
	return nil
}

func (d *DummyDBAdapter) RemoveIDToken(ctx context.Context, token OauthToken) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	_, found := d.idTokens[token.ID]
	if !found {
		return nil
	}
	delete(d.idTokens, token.ID)
	return nil
}

