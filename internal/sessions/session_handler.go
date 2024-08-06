package sessions

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type SessionHandler struct {
	cookieTemplate func() http.Cookie
	sessionMaker   SessionMaker
	sessionStore   SessionStore2
	tokenStore     TokenStore2
}

func (sh *SessionHandler) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			slog.Info("SessionHandler: before")
			session, loadErr := sh.Get(c)
			if loadErr != nil {
				slog.Info(
					"SESSION MIDDLEWARE",
					"message",
					"could not load session",
					"error",
					loadErr,
					"requestID",
					c.Response().Header().Get(echo.HeaderXRequestID),
				)
			} else {
				slog.Info(
					"SESSION MIDDLEWARE",
					"message",
					"session print",
					"session",
					session.ID,
					"sessionData",
					session,
					"requestID",
					c.Response().Header().Get(echo.HeaderXRequestID),
				)
			}
			c.Set(SessionCtxKey, session)
			err := next(c)
			slog.Info("SessionHandler: after")
			saveErr := sh.Save(c)
			if saveErr != nil {
				slog.Info(
					"SESSION MIDDLEWARE",
					"message",
					"could not save session",
					"error",
					saveErr,
					"sessionID",
					session.ID,
					"requestID",
					c.Response().Header().Get(echo.HeaderXRequestID),
				)
			}
			slog.Info(
				"SESSION MIDDLEWARE",
				"message",
				"session print",
				"session",
				session.ID,
				"ExpiresAt",
				session.ExpiresAt,
				"sessionData",
				session,
				"requestID",
				c.Response().Header().Get(echo.HeaderXRequestID),
			)
			return err
		}
	}
}

// GetFromContext retrieves a session from the current context
func (sh *SessionHandler) GetFromContext(key string, c echo.Context) (*Session, error) {
	sessionRaw := c.Get(key)
	if sessionRaw != nil {
		session, ok := sessionRaw.(*Session)
		if !ok {
			return nil, gwerrors.ErrSessionParse
		}
		if session.Expired() {
			return nil, gwerrors.ErrSessionExpired
		}
		return session, nil
	}
	return nil, gwerrors.ErrSessionNotFound
}

func (sh *SessionHandler) Get(c echo.Context) (*Session, error) {
	// check if the session is already in the request context
	session, err := sh.GetFromContext(SessionCtxKey, c)
	if err == nil {
		return session, nil
	}

	var sessionID string = ""
	// check if the session ID is in the cookie
	cookie, err := c.Cookie(SessionCookieName)
	if err != nil {
		if err == http.ErrNoCookie {
			return nil, gwerrors.ErrSessionNotFound
		}
		return nil, err
	}
	sessionID = cookie.Value

	// load the session from the store
	sessionFromStore, err := sh.sessionStore.GetSession(c.Request().Context(), sessionID)
	if err != nil {
		if err == redis.Nil {
			return nil, gwerrors.ErrSessionNotFound
		} else {
			return nil, err
		}
	}
	session = &sessionFromStore
	if session.Expired() {
		return nil, gwerrors.ErrSessionExpired
	}
	session.Touch()
	return session, nil
}

func (sh *SessionHandler) Create(c echo.Context) (*Session, error) {
	session, err := sh.sessionMaker.NewSession()
	if err != nil {
		return nil, err
	}
	c.Set(SessionCtxKey, &session)
	cookie := sh.Cookie(session)
	c.SetCookie(&cookie)
	return &session, nil
}

func (sh *SessionHandler) GetOrCreate(c echo.Context) (*Session, error) {
	session, err := sh.Get(c)
	if err != nil {
		switch err {
		case gwerrors.ErrSessionExpired:
			return sh.Create(c)
		case gwerrors.ErrSessionNotFound:
			return sh.Create(c)
		default:
			return nil, err
		}
	}
	return session, nil
}

func (sh *SessionHandler) Save(c echo.Context) error {
	session, err := sh.Get(c)
	if err != nil {
		return err
	}
	err = sh.sessionStore.SetSession(c.Request().Context(), *session)
	return err
}

func (sh *SessionHandler) Cookie(session Session) http.Cookie {
	cookie := sh.cookieTemplate()
	cookie.Value = session.ID
	return cookie
}

type SessionHandlerOption func(*SessionHandler) error

func WithSessionStore(store SessionStore2) SessionHandlerOption {
	return func(sh *SessionHandler) error {
		sh.sessionStore = store
		return nil
	}
}

func WithTokenStore(store TokenStore2) SessionHandlerOption {
	return func(sh *SessionHandler) error {
		sh.tokenStore = store
		return nil
	}
}

func WithConfig(c config.SessionConfig) SessionHandlerOption {
	return func(sh *SessionHandler) error {
		sh.sessionMaker = NewSessionMaker(WithIdleSessionTTLSeconds(c.IdleSessionTTLSeconds), WithMaxSessionTTLSeconds(c.MaxSessionTTLSeconds))
		return nil
	}
}

func NewSessionHandler(options ...SessionHandlerOption) (SessionHandler, error) {
	sh := SessionHandler{
		cookieTemplate: func() http.Cookie {
			return http.Cookie{
				Name:     SessionCookieName,
				Path:     "/",
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode}
		},
	}
	for _, opt := range options {
		opt(&sh)
	}
	if sh.cookieTemplate == nil {
		return SessionHandler{}, fmt.Errorf("cookie template is not initialized")
	}
	if sh.sessionMaker == nil {
		return SessionHandler{}, fmt.Errorf("session maker is not initialized")
	}
	if sh.sessionStore == nil {
		return SessionHandler{}, fmt.Errorf("session store is not initialized")
	}
	if sh.tokenStore == nil {
		return SessionHandler{}, fmt.Errorf("token store is not initialized")
	}
	return sh, nil
}
