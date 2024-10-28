package config

import "fmt"

type SessionConfig struct {
	IdleSessionTTLSeconds  int
	MaxSessionTTLSeconds   int
	AuthorizationVerifiers []AuthorizationVerifier
	CookieEncodingKey      RedactedString
	CookieHashKey          RedactedString
	// NOTE: UnsafeNoCookieHandler should only be used for testing, in production this has to be false/unset
	// without this there is no CSRF protection on the oauth callback endpoint
	UnsafeNoCookieHandler bool
}

type AuthorizationVerifier struct {
	Issuer          string
	Audience        string
	AuthorizedParty string
}

func (c *SessionConfig) Validate(e RunningEnvironment) error {
	if c.IdleSessionTTLSeconds <= 0 {
		return fmt.Errorf("idle session TTL seconds (%d) needs to be greater than 0", c.IdleSessionTTLSeconds)
	}
	if c.MaxSessionTTLSeconds > 0 && c.IdleSessionTTLSeconds > c.MaxSessionTTLSeconds {
		return fmt.Errorf("max session TTL seconds (%d) cannot be less than idle session TTL seconds (%d)", c.MaxSessionTTLSeconds, c.IdleSessionTTLSeconds)
	}
	if e != Development && c.UnsafeNoCookieHandler {
		return fmt.Errorf("a cookie handler needs to be configured in production")
	}
	return nil
}
