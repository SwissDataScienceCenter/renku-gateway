package config

import "fmt"

type SessionConfig struct {
	IdleSessionTTLSeconds  int
	MaxSessionTTLSeconds   int
	AuthorizationVerifiers []AuthorizationVerifier
}

type AuthorizationVerifier struct {
	Issuer          string
	Audience        string
	AuthorizedParty string
}

func (c *SessionConfig) Validate() error {
	if c.IdleSessionTTLSeconds <= 0 {
		return fmt.Errorf("idle session TTL seconds (%d) needs to be greater than 0", c.IdleSessionTTLSeconds)
	}
	if c.MaxSessionTTLSeconds > 0 && c.IdleSessionTTLSeconds > c.MaxSessionTTLSeconds {
		return fmt.Errorf("max session TTL seconds (%d) cannot be less than idle session TTL seconds (%d)", c.MaxSessionTTLSeconds, c.IdleSessionTTLSeconds)
	}
	return nil
}
