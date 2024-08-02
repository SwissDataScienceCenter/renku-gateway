package sessions

import (
	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/labstack/echo/v4"
)

func GetSession(key string, c echo.Context) (Session, error) {
	sessionRaw := c.Get(key)
	if sessionRaw != nil {
		session, ok := sessionRaw.(Session)
		if !ok {
			return Session{}, gwerrors.ErrSessionParse
		}
		if session.Expired() {
			return Session{}, gwerrors.ErrSessionExpired
		}
		return session, nil
	}
	return Session{}, gwerrors.ErrSessionNotFound
}
