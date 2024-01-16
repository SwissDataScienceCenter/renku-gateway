package models

import (
	"fmt"
	"time"
)

// OauthToken is a struct used to store and work with Oauth access and refresh tokens
type OauthToken struct {
	ID         string
	Value      string
	ExpiresAt  time.Time
	TokenURL   string
	ProviderID string
	Type       OauthTokenType
	encryptor  Encryptor
}

// SetEncryptor adds encryption capabilities to the token
func (o OauthToken) SetEncryptor(enc Encryptor) OauthToken {
	output := o
	output.encryptor = enc
	return output
}

// Encrypt encrypts the value of the token if an encryptor is set
func (o OauthToken) Encrypt() (OauthToken, error) {
	if o.encryptor == nil {
		return o, nil
	}
	encValue, err := o.encryptor.Encrypt(o.Value)
	if err != nil {
		return OauthToken{}, err
	}
	output := o
	output.Value = encValue
	return output, nil
}

// Decrypt decrypts the value of the token if an encryptor is set
func (o OauthToken) Decrypt() (OauthToken, error) {
	if o.encryptor == nil {
		return o, nil
	}
	decValue, err := o.encryptor.Decrypt(o.Value)
	if err != nil {
		return OauthToken{}, err
	}
	output := o
	output.Value = decValue
	return output, nil
}

// String immplements the Stringer interface for printing the token in logs
func (o OauthToken) String() string {
	return fmt.Sprintf(
		"%s<ID: %s, Value: redacted, ExpiresAt: %s, TokenURL: %s, ProviderID: %s, Encryption: %v>",
		o.Type,
		o.ID,
		o.ExpiresAt,
		o.TokenURL,
		o.ProviderID,
		o.encryptor != nil,
	)
}

func (o OauthToken) Expired() bool {
	return time.Now().UTC().After(o.ExpiresAt)
}
