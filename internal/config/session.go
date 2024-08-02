package config

import "fmt"

type SessionConfig struct {
	IdleSessionTTLSeconds int
	MaxSessionTTLSeconds  int
}

func (c *SessionConfig) Validate() error {
	if c.MaxSessionTTLSeconds > 0 && c.IdleSessionTTLSeconds > c.MaxSessionTTLSeconds {
		return fmt.Errorf("max session TTL seconds (%d) cannot be less than idle session TTL seconds (%d)", c.MaxSessionTTLSeconds, c.IdleSessionTTLSeconds)
	}
	return nil
}
