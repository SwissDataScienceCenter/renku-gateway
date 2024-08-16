package sessions

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/authentication"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/utils"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type SessionStore struct {
	authenticator  authentication.Authenticator
	cookieTemplate func() http.Cookie
	sessionMaker   SessionMaker
	sessionRepo    models.SessionRepository
	tokenStore     models.TokenStoreInterface
}

func (sessions *SessionStore) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			session, loadErr := sessions.Get(c)
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
			saveErr := sessions.Save(c)
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
			session, _ = sessions.getFromContext(c)
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

// Get returns the current session
func (sessions *SessionStore) Get(c echo.Context) (*models.Session, error) {
	// check if the session is already in the request context
	session, err := sessions.getFromContext(c)
	if err == nil {
		return session, nil
	}
	// check if the session ID is in the cookie
	sessionID, err := sessions.getSessionIDFromCookie(c)
	if err != nil {
		return &models.Session{}, err
	}
	// check if we can create a session from headers or basic auth
	if sessionID == "" {
		session, err = sessions.getFromHeaders(c)
		if err == nil {
			return session, nil
		}
		session, err = sessions.getFromBasicAuth(c)
		if err == nil {
			return session, nil
		}
	}
	// No session found
	if sessionID == "" {
		return &models.Session{}, gwerrors.ErrSessionNotFound
	}
	// load the session from the store
	sessionFromStore, err := sessions.sessionRepo.GetSession(c.Request().Context(), sessionID)
	if err != nil {
		if err == redis.Nil {
			return &models.Session{}, gwerrors.ErrSessionNotFound
		} else {
			return &models.Session{}, err
		}
	}
	session = &sessionFromStore
	if session.Expired() {
		return &models.Session{}, gwerrors.ErrSessionExpired
	}
	session.Touch()
	return session, nil
}

// Create will create a new session.
func (sessions *SessionStore) Create(c echo.Context) (*models.Session, error) {
	session, err := sessions.sessionMaker.NewSession()
	if err != nil {
		return &models.Session{}, err
	}
	c.Set(SessionCtxKey, &session)
	cookie := sessions.Cookie(session)
	c.SetCookie(&cookie)
	return &session, nil
}

func (sessions *SessionStore) Save(c echo.Context) error {
	session, err := sessions.Get(c)
	if err != nil {
		return err
	}
	// NOTE: ephemeral session, do not save
	if session.ID == "" {
		return nil
	}
	return sessions.sessionRepo.SetSession(c.Request().Context(), *session)
}

// Delete removes the current session from storage and unsets the session cookie
func (sessions *SessionStore) Delete(c echo.Context) error {
	sessionID, err := sessions.getSessionIDFromCookie(c)
	if err != nil {
		return err
	}

	newCookie := sessions.cookieTemplate()
	newCookie.MaxAge = -1
	c.SetCookie(&newCookie)

	c.Set(SessionCtxKey, &models.Session{})

	if sessionID == "" {
		return nil
	}
	return sessions.sessionRepo.RemoveSession(c.Request().Context(), sessionID)
}

func (sessions *SessionStore) Cookie(session models.Session) http.Cookie {
	cookie := sessions.cookieTemplate()
	cookie.Value = session.ID
	return cookie
}

// getSessionIDFromCookie returns the session ID from the cookie if present
func (sessions *SessionStore) getSessionIDFromCookie(c echo.Context) (string, error) {
	sessionID := ""
	cookie, err := c.Cookie(SessionCookieName)
	if err != nil {
		if err != http.ErrNoCookie {
			return "", nil
		}
	} else {
		sessionID = cookie.Value
	}
	return sessionID, nil
}

// getFromContext retrieves a session from the current context
func (sessions *SessionStore) getFromContext(c echo.Context) (*models.Session, error) {
	sessionRaw := c.Get(SessionCtxKey)
	if sessionRaw != nil {
		session, ok := sessionRaw.(*models.Session)
		if session == nil {
			return &models.Session{}, gwerrors.ErrSessionNotFound
		}
		if !ok {
			return &models.Session{}, gwerrors.ErrSessionParse
		}
		if session.Expired() {
			return &models.Session{}, gwerrors.ErrSessionExpired
		}
		return session, nil
	}
	return &models.Session{}, gwerrors.ErrSessionNotFound
}

// getFromHeaders creates a session from the Authorization header if present
func (sessions *SessionStore) getFromHeaders(c echo.Context) (*models.Session, error) {
	accessToken := c.Request().Header.Get(echo.HeaderAuthorization)
	accessToken = strings.TrimPrefix(accessToken, "Bearer ")
	accessToken = strings.TrimPrefix(accessToken, "bearer ")
	if accessToken != "" {
		claims, err := sessions.authenticator.VerifyAccessToken(c.Request().Context(), accessToken)
		if err == nil {
			userID := claims.Subject
			tokenIDs := map[string]string{"renku": "renku:" + userID, "gitlab": "gitlab:" + userID}
			// make an ephemeral session
			session := models.Session{
				CreatedAt: time.Now().UTC(),
				UserID:    userID,
				TokenIDs:  tokenIDs,
			}
			c.Set(SessionCtxKey, &session)
			// remove the authorization header, it will be re-populated if needed
			c.Request().Header.Del(echo.HeaderAuthorization)
			return &session, nil
		}
	}
	return &models.Session{}, gwerrors.ErrSessionNotFound
}

// getFromBasicAuth creates a session from basic authorization
func (sessions *SessionStore) getFromBasicAuth(c echo.Context) (*models.Session, error) {
	_, basicAuthPwd, ok := c.Request().BasicAuth()
	if ok {
		claims, err := sessions.authenticator.VerifyAccessToken(c.Request().Context(), basicAuthPwd)
		if err == nil {
			userID := claims.Subject
			tokenIDs := map[string]string{"renku": "renku:" + userID, "gitlab": "gitlab:" + userID}
			// make an ephemeral session
			session := models.Session{
				CreatedAt: time.Now().UTC(),
				UserID:    userID,
				TokenIDs:  tokenIDs,
			}
			c.Set(SessionCtxKey, &session)
			// remove the authorization header, it will be re-populated if needed
			c.Request().Header.Del(echo.HeaderAuthorization)
			return &session, nil
		}
	}
	return &models.Session{}, gwerrors.ErrSessionNotFound
}

type SessionStoreOption func(*SessionStore) error

func WithAuthenticator(a authentication.Authenticator) SessionStoreOption {
	return func(sessions *SessionStore) error {
		sessions.authenticator = a
		return nil
	}
}

func WithSessionRepository(repo models.SessionRepository) SessionStoreOption {
	return func(sessions *SessionStore) error {
		sessions.sessionRepo = repo
		return nil
	}
}

func WithTokenStore(store models.TokenStoreInterface) SessionStoreOption {
	return func(sessions *SessionStore) error {
		sessions.tokenStore = store
		return nil
	}
}

func WithConfig(c config.SessionConfig) SessionStoreOption {
	return func(sessions *SessionStore) error {
		sessions.sessionMaker = NewSessionMaker(WithIdleSessionTTLSeconds(c.IdleSessionTTLSeconds), WithMaxSessionTTLSeconds(c.MaxSessionTTLSeconds))
		return nil
	}
}

func NewSessionStore(options ...SessionStoreOption) (*SessionStore, error) {
	sessions := SessionStore{
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
		opt(&sessions)
	}
	if sessions.authenticator == nil {
		return &SessionStore{}, fmt.Errorf("authenticator is not initialized")
	}
	if sessions.cookieTemplate == nil {
		return &SessionStore{}, fmt.Errorf("cookie template is not initialized")
	}
	if sessions.sessionMaker == nil {
		return &SessionStore{}, fmt.Errorf("session maker is not initialized")
	}
	if sessions.sessionRepo == nil {
		return &SessionStore{}, fmt.Errorf("session repository is not initialized")
	}
	if sessions.tokenStore == nil {
		return &SessionStore{}, fmt.Errorf("token store is not initialized")
	}
	return &sessions, nil
}
