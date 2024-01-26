package models

import (
	"fmt"
	"net/http"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)


type SessionHandlerOption func(*SessionHandler)

type SessionHandler struct {
	cookieTemplate           func() http.Cookie
	sessionTTL               time.Duration
	tokenStore               TokenStore
	sessionStore             SessionStore
	createSessionIfMissing   bool
	recreateSessionIfExpired bool
	contextKey               string
	headerKey                string
}

func (s *SessionHandler) cookie(session *Session) *http.Cookie {
	if session == nil {
		return nil
	}
	if session.Expired() {
		return nil
	}
	cookie := s.cookieTemplate()
	cookie.Value = session.ID
	cookie.Expires = session.CreatedAt.Add(session.TTL())
	return &cookie
}

func (s *SessionHandler) Remove(c echo.Context) error {
	if s.sessionStore == nil {
		return fmt.Errorf("cannot remove a session when the session store is not defined")
	}
	sessionIDs := mapset.NewSet[string]() 
	sessionID := c.Request().Header.Get(s.headerKey)
	// remove the request header if set
	if sessionID != "" {
		c.Request().Header.Del(s.headerKey)
		sessionIDs.Add(sessionID)
	}
	sessionID = c.Response().Header().Get(s.headerKey)
	// remove the response header if set
	if sessionID != "" {
		c.Response().Header().Del(s.headerKey)
		sessionIDs.Add(sessionID)
	}
	cookie, err := c.Cookie(s.cookieTemplate().Name)
	if err != nil && err != http.ErrNoCookie {
		return err
	}
	// remove the cookie if present
	if cookie != nil {
		sessionIDs.Add(cookie.Value)
		c.SetCookie(&http.Cookie{Name: s.cookieTemplate().Name, Value: "", MaxAge: -1})
	}
	// remove session from the context if present
	if c.Get(s.contextKey) != nil {
		c.Set(s.contextKey, nil)
	}
	for _, id := range sessionIDs.ToSlice() {
		err = s.sessionStore.RemoveSession(c.Request().Context(), id)
		if err == redis.Nil {
			// the session is not in the store - we ignore this
			err = nil
		}
	}
	return err 
}

func (s *SessionHandler) load(c echo.Context) (Session, error) {
	if s.sessionStore == nil {
		return Session{}, fmt.Errorf("cannot load a session when the session store is not defined")
	}
	// check if the session is already in the request context
	sessionRaw := c.Get(s.contextKey)
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
	// check if the session ID is in the request header
	sessionID := c.Request().Header.Get(s.headerKey)
	if sessionID == "" {
		// check if the session ID is in the cookie
		cookie, err := c.Cookie(s.cookieTemplate().Name)
		if err != nil {
			if err == http.ErrNoCookie {
				return Session{}, gwerrors.ErrSessionNotFound
			}
			return Session{}, err
		}
		sessionID = cookie.Value
	}
	// load the session from the store
	session, err := s.sessionStore.GetSession(c.Request().Context(), sessionID)
	if err != nil {
		if err == redis.Nil {
			return Session{}, gwerrors.ErrSessionNotFound
		} else {
			return Session{}, err
		}
	}
	if session.Expired() {
		return Session{}, gwerrors.ErrSessionExpired
	}
	session.sessionStore = s.sessionStore
	session.tokenStore = s.tokenStore
	return session, nil
}

func (s *SessionHandler) create(c echo.Context) (Session, error) {
	session, err := NewSession(SessionWithTokenStore(s.tokenStore), SessionWithSessionStore(s.sessionStore))
	if err != nil {
		return Session{}, err
	}
	err = session.Save(c.Request().Context())
	if err != nil {
		return Session{}, err
	}
	c.Set(s.contextKey, session)
	c.Request().Header.Set(s.headerKey, session.ID)
	c.SetCookie(s.cookie(&session))
	return session, nil 
}

func (s *SessionHandler) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			session, err := s.load(c)
			if err != nil {
				switch err {
				case gwerrors.ErrSessionExpired:
					if !s.recreateSessionIfExpired {
						return next(c)
					}
					_, err := s.create(c)
					if err != nil {
						return err
					}
					return next(c)
				case gwerrors.ErrSessionNotFound:
					if !s.createSessionIfMissing {
						return next(c)
					}
					_, err := s.create(c)
					if err != nil {
						return err
					}
					return next(c)
				default:
					return err
				}
			}
			c.Set(s.contextKey, session)
			return next(c)
		}
	}
}

func WithSessionTTL(ttl time.Duration) SessionHandlerOption {
	return func(s *SessionHandler) {
		s.sessionTTL = ttl
	}
}

// Note that the value of the cookie and expiry will be rewritten when generated
// also the cookie name will always come from the config constant and will be ignored if set in the template
func WithCookieTemplate(cookie http.Cookie) SessionHandlerOption {
	return func(s *SessionHandler) {
		s.cookieTemplate = func() http.Cookie {
			cookie.Name = SessionCookieName
			return cookie
		}
	}
}

func WithSessionStore(store SessionStore) SessionHandlerOption {
	return func(s *SessionHandler) {
		s.sessionStore = store
	}
}

func WithTokenStore(store TokenStore) SessionHandlerOption {
	return func(s *SessionHandler) {
		s.tokenStore = store
	}
}

func DontCreateIfMissing() SessionHandlerOption {
	return func(s *SessionHandler) {
		s.createSessionIfMissing = false
	}
}

func DontRecreateIfExpired() SessionHandlerOption {
	return func(s *SessionHandler) {
		s.recreateSessionIfExpired = false
	}
}

func NewSessionHandler(options ...SessionHandlerOption) SessionHandler {
	store := NewDummyDBAdapter()
	sh := SessionHandler{
		recreateSessionIfExpired: true,
		createSessionIfMissing:   true,
		sessionTTL:               time.Hour,
		tokenStore:               &store,
		sessionStore:             &store,
		contextKey:               SessionCtxKey,
		headerKey:                SessionHeaderKey,
		cookieTemplate: func() http.Cookie {
			return http.Cookie{
				Name:     SessionCookieName,
				Secure:   false,
				HttpOnly: true,
				Path:     "/",
				MaxAge:   3600,
			}
		},
	}
	for _, opt := range options {
		opt(&sh)
	}
	return sh
}
