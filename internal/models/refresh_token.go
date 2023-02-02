package models

import "time"

type RefreshToken struct {
	ID        string
	Value     string
	ExpiresAt time.Time
}
