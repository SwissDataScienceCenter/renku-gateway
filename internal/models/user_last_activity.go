package models

import "time"

// UserLastActivity represents a user's last activity (as seen by the gateway)
type UserLastActivity struct {
	// ID of the user
	UserID string
	// UTC timestamp of the user's last activity
	LastActivity time.Time
	// UTC timestamp for when this record will expire
	ExpiresAt time.Time
}

func (u *UserLastActivity) Expired() bool {
	if u.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().UTC().After(u.ExpiresAt)
}

// Touch updates the LastActivity and ExpiresAt fields
func (u *UserLastActivity) Touch() {
	u.LastActivity = time.Now()
	// TODO: configure user last activity idle time
	u.ExpiresAt = u.LastActivity.Add(190 * 24 * time.Hour)
}
