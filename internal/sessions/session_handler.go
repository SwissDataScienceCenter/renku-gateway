package sessions

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/utils"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type SessionHandler struct {
	cookieTemplate func() http.Cookie
	sessionMaker   SessionMaker
	sessionStore   SessionRepository
	tokenRefresher TokenRefresher
	tokenStore     SessionTokenRepository
}

func (sh *SessionHandler) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			session, loadErr := sh.Get(c)
			if loadErr != nil && loadErr != gwerrors.ErrSessionNotFound && loadErr != gwerrors.ErrSessionExpired {
				slog.Info(
					"SESSION MIDDLEWARE",
					"message",
					"could not load session",
					"error",
					loadErr,
					"requestID",
					utils.GetRequestID(c),
				)
			}
			slog.Debug(
				"SESSION MIDDLEWARE",
				"message",
				"session print (before)",
				"session",
				session,
				"requestID",
				utils.GetRequestID(c),
			)
			c.Set(SessionCtxKey, session)
			err := next(c)
			saveErr := sh.Save(c)
			if saveErr != nil && saveErr != gwerrors.ErrSessionNotFound && saveErr != gwerrors.ErrSessionExpired {
				sessionID := ""
				if session != nil {
					sessionID = session.ID
				}
				slog.Info(
					"SESSION MIDDLEWARE",
					"message",
					"could not save session",
					"error",
					saveErr,
					"sessionID",
					sessionID,
					"requestID",
					utils.GetRequestID(c),
				)
			}
			session, _ = sh.Get(c)
			slog.Debug(
				"SESSION MIDDLEWARE",
				"message",
				"session print (after)",
				"session",
				session,
				"requestID",
				utils.GetRequestID(c),
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
		if session == nil {
			return &Session{}, gwerrors.ErrSessionNotFound
		}
		if !ok {
			return &Session{}, gwerrors.ErrSessionParse
		}
		if session.Expired() {
			return &Session{}, gwerrors.ErrSessionExpired
		}
		return session, nil
	}
	return &Session{}, gwerrors.ErrSessionNotFound
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
			return &Session{}, gwerrors.ErrSessionNotFound
		}
		return &Session{}, err
	}
	sessionID = cookie.Value

	// load the session from the store
	sessionFromStore, err := sh.sessionStore.GetSession(c.Request().Context(), sessionID)
	if err != nil {
		if err == redis.Nil {
			return &Session{}, gwerrors.ErrSessionNotFound
		} else {
			return &Session{}, err
		}
	}
	session = &sessionFromStore
	if session.Expired() {
		return &Session{}, gwerrors.ErrSessionExpired
	}
	session.Touch()
	return session, nil
}

// Create will create a new session.
func (sh *SessionHandler) Create(c echo.Context) (*Session, error) {
	session, err := sh.sessionMaker.NewSession()
	if err != nil {
		return &Session{}, err
	}
	c.Set(SessionCtxKey, &session)
	cookie := sh.Cookie(session)
	c.SetCookie(&cookie)
	return &session, nil
}

func (sh *SessionHandler) Save(c echo.Context) error {
	session, err := sh.Get(c)
	if err != nil {
		return err
	}
	return sh.sessionStore.SetSession(c.Request().Context(), *session)
}

func (sh *SessionHandler) Cookie(session Session) http.Cookie {
	cookie := sh.cookieTemplate()
	cookie.Value = session.ID
	return cookie
}

type SessionHandlerOption func(*SessionHandler) error

func WithSessionStore(store SessionRepository) SessionHandlerOption {
	return func(sh *SessionHandler) error {
		sh.sessionStore = store
		return nil
	}
}

func WithTokenRefresher(tr TokenRefresher) SessionHandlerOption {
	return func(sh *SessionHandler) error {
		sh.tokenRefresher = tr
		return nil
	}
}

func WithTokenStore(store SessionTokenRepository) SessionHandlerOption {
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
	if sh.tokenRefresher == nil {
		return SessionHandler{}, fmt.Errorf("token refresher is not initialized")
	}
	if sh.tokenStore == nil {
		return SessionHandler{}, fmt.Errorf("token store is not initialized")
	}
	return sh, nil
}
