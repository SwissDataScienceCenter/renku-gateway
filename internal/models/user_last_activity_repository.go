package models

import (
	"context"
)

// UserLastActivityRepository represents the interface used to persist users' last activities
type UserLastActivityRepository interface {
	UserLastActivityGetter
	UserLastActivitySetter
}

type UserLastActivityGetter interface {
	GetUserLastActivity(ctx context.Context, userID string) (UserLastActivity, error)
}

type UserLastActivitySetter interface {
	SetUserLastActivity(ctx context.Context, userLastActivity UserLastActivity) error
}
