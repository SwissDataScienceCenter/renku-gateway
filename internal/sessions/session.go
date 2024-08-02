package sessions

import (
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

// Session represents a persistent session between a client and the gateway
type Session struct {
	ID string
	// UTC timestamp for when the session was created
	CreatedAt time.Time
	// UTC timestamp for when the session will expire
	ExpiresAt      time.Time
	IdleTTLSeconds models.SerializableInt
	MaxTTLSeconds  models.SerializableInt
}

func (s *Session) Expired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}

// Touch() updates a session's ExpiresAt field according to IdleTTLSeconds and MaxTTLSeconds
func (s *Session) Touch() {
	maxExpiresAt := s.CreatedAt.Add(s.MaxTTL())
	expiresAt := time.Now().UTC().Add(s.IdleTTL())
	if expiresAt.After(maxExpiresAt) {
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
