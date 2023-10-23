package models

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"time"
)

type Session struct {
	ID                 string
	Type               SessionType
	ExpiresAt          time.Time
	TokenIDs           SerializableStringSlice
	LoginWithProviders SerializableStringSlice
	// The url to redirect to when the login flow is complete (i.e. Renku homepage)
	RedirectURL string
}

func (s *Session) Expired() bool {
	return time.Now().After(s.ExpiresAt)
}

func (s *Session) PopProviderID() string {
	if len(s.LoginWithProviders) == 0 {
		return ""
	}
	output := s.LoginWithProviders[0]
	s.LoginWithProviders = append(SerializableStringSlice{}, s.LoginWithProviders[1:]...)
	return output
}

func (s *Session) PeekProviderID() string {
	if len(s.LoginWithProviders) == 0 {
		return ""
	}
	return s.LoginWithProviders[0]
}

func (s *Session) AddTokenID(id string) {
	s.TokenIDs = append(s.TokenIDs, id)
}

func (s *Session) SetRedirectURL(url string) {
	s.RedirectURL = url
}

func (s *Session) SetProviderIDs(ids []string) {
	s.LoginWithProviders = ids
}

func (s *Session) Cookie(name, valuePrefix string, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    valuePrefix + s.ID,
		Path:     "/",
		Expires:  time.Now().Add(time.Hour * 24 * 7),
		HttpOnly: true,
		Secure:   secure,
	}
}

func NewSession(ttl time.Duration, providers SerializableStringSlice) (Session, error) {
	now := time.Now()
	id, err := randString(24)
	if err != nil {
		return Session{}, err
	}
	return Session{
		ID:                 id,
		ExpiresAt:          now.Add(ttl),
		LoginWithProviders: providers,
	}, nil
}

func randString(nByte int) (string, error) {
	b := make([]byte, nByte)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
