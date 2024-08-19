package models

import (
	"time"
)

var randomStateGenerator IDGenerator = RandomGenerator{Length: 24}

// Session represents a persistent session between a client and the gateway
type Session struct {
	ID string
	// UTC timestamp for when the session was created
	CreatedAt time.Time
	// UTC timestamp for when the session will expire
	ExpiresAt      time.Time
	IdleTTLSeconds SerializableInt
	MaxTTLSeconds  SerializableInt
	UserID         string
	// Map of providerID to tokenID
	TokenIDs SerializableMap
	// The url to redirect to when the login flow is complete (i.e. Renku homepage)
	LoginRedirectURL string
	// The sequence of providers for the login flow
	LoginSequence SerializableStringSlice
	// State value used during login flows
	LoginState string
}

func (s *Session) Expired() bool {
	if s.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().UTC().After(s.ExpiresAt)
}

// Touch() updates a session's ExpiresAt field according to IdleTTLSeconds and MaxTTLSeconds
func (s *Session) Touch() {
	if s.IdleTTLSeconds == 0 && s.MaxTTLSeconds == 0 {
		s.ExpiresAt = time.Time{}
		return
	} else if s.IdleTTLSeconds == 0 {
		s.ExpiresAt = s.CreatedAt.Add(s.MaxTTL())
		return
	}
	maxExpiresAt := s.CreatedAt.Add(s.MaxTTL())
	expiresAt := time.Now().UTC().Add(s.IdleTTL())
	if s.MaxTTLSeconds > 0 && expiresAt.After(maxExpiresAt) {
		expiresAt = maxExpiresAt
	}
	s.ExpiresAt = expiresAt
}

func (s *Session) IdleTTL() time.Duration {
	return time.Duration(s.IdleTTLSeconds) * time.Second
}

func (s *Session) MaxTTL() time.Duration {
	return time.Duration(s.MaxTTLSeconds) * time.Second
}

func (s *Session) GenerateLoginState() error {
	state, err := randomStateGenerator.ID()
	if err != nil {
		return err
	}
	s.LoginState = state
	return nil
}
