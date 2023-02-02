package models

import "time"

type AccessToken struct {
	ID        string
	Value     string
	ExpiresAt time.Time
	URL       string
	Type      string
}
