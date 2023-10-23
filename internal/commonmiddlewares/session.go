package commonmiddlewares

import (
	"fmt"
	"net/http"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/commonconfig"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/labstack/echo/v4"
)

type sessionStore interface {
	models.SessionGetter
	models.SessionSetter
	models.SessionRemover
}

// SessionMiddleware ensures that there is always a session cookie and a corresponding
// session in the store. NOTE: the cookie is untrusted so the cookie should always reflect
// the state of the store - NEVER the other way around.
type SessionMiddleware struct {
	store        sessionStore
	cookieName   string
	secureCookie bool
}

// Middleware generates an echo middleware that will check for the session cookie
// and the session store and if necessary generate first the session and then the cookie.
func (s *SessionMiddleware) Middleware(sessionType models.SessionType) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var err error
			var sessionCookie *http.Cookie
			var session models.Session
			var sessionID string
			sessionCookie, err = c.Request().Cookie(s.cookieName)
			authHeader := c.Request().Header.Get(http.CanonicalHeaderKey("authorization"))
			if err == http.ErrNoCookie && len(authHeader) > 7 {
				// if the session cookie does not exist look in the Authorization header
				sessionID = authHeader[7:]
			}
			if err != http.ErrNoCookie && err != nil {
				return fmt.Errorf("getting session cookie %s failed: %v", s.cookieName, err)
			}
			if sessionID == "" && sessionCookie == nil {
				// the session cookie does not exist
				session, err = s.newSession(c, commonconfig.SessionTTL()[sessionType])
				if err != nil {
					return err
				}
				c.Set(commonconfig.SessionIDCtxKey, session.ID)
				c.Set(commonconfig.SessionCtxKey, session)
				return next(c)
			}
			if sessionID == "" && sessionCookie != nil {
				sessionID = sessionCookie.Value
			}
			// check if the session is present in the store
			session, err = s.store.GetSession(c.Request().Context(), sessionID)
			if err != nil {
				return err
			}
			// no session in the store or the session is expired
			if session.ID != "" && session.Expired() {
				// remove the expired session
				err = s.store.RemoveSession(c.Request().Context(), session.ID)
			}
			if session.ID == "" || session.Expired() {
				// make a new session
				session, err = s.newSession(c, commonconfig.SessionTTL()[sessionType])
				if err != nil {
					return err
				}
			}
			// add the session and sessionID in the context
			c.Set(commonconfig.SessionIDCtxKey, session.ID)
			c.Set(commonconfig.SessionCtxKey, session)
			return next(c)
		}
	}
}

// newSession makes a new session, saves it in the store and adds the session ID
// in a cookie in the request and in the response
func (s *SessionMiddleware) newSession(c echo.Context, sessionTTL time.Duration) (models.Session, error) {
	session, err := models.NewSession(sessionTTL, models.SerializableStringSlice{})
	if err != nil {
		return models.Session{}, err
	}
	err = s.store.SetSession(c.Request().Context(), session)
	if err != nil {
		return models.Session{}, err
	}
	sessionCookie := session.Cookie(s.cookieName, "", s.secureCookie)
	c.Request().AddCookie(sessionCookie)
	c.SetCookie(sessionCookie)
	return session, nil
}

func NewSessionMiddleware(store sessionStore, cookieName string, secureCookie bool) *SessionMiddleware {
	return &SessionMiddleware{
		store:        store,
		cookieName:   cookieName,
		secureCookie: secureCookie,
	}
}
