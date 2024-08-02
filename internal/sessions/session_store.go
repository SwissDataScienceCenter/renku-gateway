package sessions

import "context"

type SessionStore2 interface {
	SessionGetter
	SessionSetter
	SessionRemover
}

type SessionGetter interface {
	GetSession(context.Context, string) (Session, error)
}

type SessionSetter interface {
	SetSession(context.Context, Session) error
}

type SessionRemover interface {
	RemoveSession(context.Context, string) error
}
