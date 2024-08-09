package models

import (
	"fmt"
	"time"
)

// AuthToken is a struct used to store and work with OAuth 2.0 access, OAuth 2.0 refresh tokens and OIDC ID tokens
type AuthToken struct {
	ID         string
	Value      string
	ExpiresAt  time.Time
	Subject    string
	TokenURL   string
	ProviderID string
	Type       OauthTokenType
	SessionID  string
	// encryptor  Encryptor
}

// // SetEncryptor adds encryption capabilities to the token
// func (o OauthToken) SetEncryptor(enc Encryptor) OauthToken {
// 	output := o
// 	output.encryptor = enc
// 	return output
// }

// Encrypt encrypts the value of the token if an encryptor is set
func (o AuthToken) Encrypt(e Encryptor) (AuthToken, error) {
	if e == nil {
		return o, nil
	}
	encValue, err := e.Encrypt(o.Value)
	if err != nil {
		return AuthToken{}, err
	}
	output := o
	output.Value = encValue
	return output, nil
}

// Decrypt decrypts the value of the token if an encryptor is set
func (o AuthToken) Decrypt(e Encryptor) (AuthToken, error) {
	if e == nil {
		return o, nil
	}
	decValue, err := e.Decrypt(o.Value)
	if err != nil {
		return AuthToken{}, err
	}
	output := o
	output.Value = decValue
	return output, nil
}

// String immplements the Stringer interface for printing the token in logs
func (o AuthToken) String() string {
	return fmt.Sprintf(
		"%s<ID: %s, Value: redacted, ExpiresAt: %s, TokenURL: %s, ProviderID: %s, SessionID: %s>",
		o.Type,
		o.ID,
		o.ExpiresAt,
		o.TokenURL,
		o.ProviderID,
		o.SessionID,
	)
}

func (o AuthToken) Expired() bool {
	if o.ExpiresAt.IsZero() {
		// IF the expiry date is missing we assume the token is not expired
		return false
	}
	return time.Now().UTC().After(o.ExpiresAt)
}

func (o AuthToken) ExpiresSoon(margin time.Duration) bool {
	if o.ExpiresAt.IsZero() {
		// IF the expiry date is missing we assume the token is not expired
		return false
	}
	return time.Now().UTC().After(o.ExpiresAt.Add(-margin))
}
