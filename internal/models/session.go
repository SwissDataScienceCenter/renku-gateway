package models

import "time"

type Session struct {
	ID        string
	Type      string
	ExpiresAt time.Time
	TokenIDs  []string
}
