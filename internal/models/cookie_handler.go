package models

// CookieHandler represents the interface used to encrypt and decrypt cookies
type CookieHandler interface {
	Encode(name string, value interface{}) (string, error)
	Decode(name, value string, dst interface{}) error
}
