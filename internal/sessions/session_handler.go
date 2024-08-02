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
	sessionMaker SessionMaker
	sessionStore SessionStore2
}

func (sh *SessionHandler) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			slog.Info("SessionHandler: before")
			session, loadErr := sh.LoadOrCreate(c)
			if loadErr != nil {
				slog.Info(
					"SESSION MIDDLEWARE",
					"message",
					"could not load session",
					"requestID",
					c.Request().Header.Get("X-Request-ID"),
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
					"sessionID",
					session.ID,
					"requestID",
					c.Request().Header.Get("X-Request-ID"),
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
				"requestID",
				c.Request().Header.Get("X-Request-ID"),
			)
			return err
		}
	}
}

func (sh *SessionHandler) Get(c echo.Context) (Session, error) {
	return GetSession(SessionCtxKey, c)
}

func (sh *SessionHandler) Create(c echo.Context) (Session, error) {
	session, err := sh.sessionMaker.NewSession()
	if err != nil {
		return Session{}, err
	}
	c.SetCookie(&http.Cookie{
		Name:     SessionCookieName,
		Secure:   true,
		HttpOnly: true,
		Path:     "/",
		Expires:  session.ExpiresAt,
	})
	return session, nil
}

func (sh *SessionHandler) Load(c echo.Context) (Session, error) {
	if sh.sessionStore == nil {
		return Session{}, fmt.Errorf("cannot load a session when the session store is not defined")
	}
	// check if the session is already in the request context
	sessionRaw := c.Get(SessionCtxKey)
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

	var sessionID string = ""
	// check if the session ID is in the cookie
	cookie, err := c.Cookie(SessionCookieName)
	if err != nil {
		if err == http.ErrNoCookie {
			return Session{}, gwerrors.ErrSessionNotFound
		}
		return Session{}, err
	}
	sessionID = cookie.Value

	// load the session from the store
	session, err := sh.sessionStore.GetSession(c.Request().Context(), sessionID)
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
	session.Touch()
	return session, nil
}

func (sh *SessionHandler) LoadOrCreate(c echo.Context) (Session, error) {
	session, err := sh.Load(c)
	if err != nil {
		switch err {
		case gwerrors.ErrSessionExpired:
			// if !sh.recreateSessionIfExpired {
			// 	return next(c)
			// }
			// _, err := s.Create(c)
			// if err != nil {
			// 	return err
			// }
			// return next(c)
			return sh.Create(c)

		case gwerrors.ErrSessionNotFound:
			// if !s.createSessionIfMissing {
			// 	return next(c)
			// }
			// _, err := s.Create(c)
			// if err != nil {
			// 	return err
			// }
			// return next(c)
			return sh.Create(c)
		default:
			return Session{}, err
		}
	}
	return session, nil
}

func (sh *SessionHandler) Save(c echo.Context) error {
	if sh.sessionStore == nil {
		return fmt.Errorf("cannot save a session when the session store is not defined")
	}
	session, err := sh.Get(c)
	if err != nil {
		return err
	}
	err = sh.sessionStore.SetSession(c.Request().Context(), session)
	return err
}

type SessionHandlerOption func(*SessionHandler) error

// func WithConfig(loginConfig config.LoginConfig) LoginServerOption {
// 	return func(l *LoginServer) error {
// 		l.config = &loginConfig
// 		providerStore, err := oidc.NewClientStore(loginConfig.Providers)
// 		if err != nil {
// 			return err
// 		}
// 		l.providerStore = providerStore
// 		return nil
// 	}
// }

func WithConfig(c config.SessionConfig, e config.RunningEnvironment) SessionHandlerOption {
	return func(sh *SessionHandler) error {
		sh.sessionMaker = NewSessionMaker(WithIdleSessionTTLSeconds(c.IdleSessionTTLSeconds), WithMaxSessionTTLSeconds(c.MaxSessionTTLSeconds))
		store := NewInMemorySessionStore()
		sh.sessionStore = &store

		// TODO: fail if in memory store and production

		return nil
	}
}

func NewSessionHandler(options ...SessionHandlerOption) (SessionHandler, error) {
	sh := SessionHandler{}
	for _, opt := range options {
		opt(&sh)
	}
	if sh.sessionMaker == nil {
		return SessionHandler{}, fmt.Errorf("session maker is not initialized")
	}
	if sh.sessionStore == nil {
		return SessionHandler{}, fmt.Errorf("session store is not initialized")
	}
	return sh, nil
}
