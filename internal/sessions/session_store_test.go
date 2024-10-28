package sessions

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/authentication"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/db"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/tokenstore"
	"github.com/gorilla/securecookie"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSessionStore(t *testing.T, options ...SessionStoreOption) *SessionStore {
	dbAdapter, err := db.NewRedisAdapter(db.WithRedisConfig(config.RedisConfig{
		Type: config.DBTypeRedisMock,
	}))
	require.NoError(t, err)
	tokenStore, err := tokenstore.NewTokenStore(
		tokenstore.WithExpiryMargin(time.Duration(3)*time.Minute),
		tokenstore.WithConfig(config.LoginConfig{}),
		tokenstore.WithTokenRepository(dbAdapter),
	)
	require.NoError(t, err)
	authenticator, err := authentication.NewAuthenticator()
	require.NoError(t, err)
	sessionStoreOptions := []SessionStoreOption{
		WithAuthenticator(authenticator),
		WithSessionRepository(dbAdapter),
		WithTokenStore(tokenStore),
		WithConfig(config.SessionConfig{
			UnsafeNoCookieHandler: true,
		}),
		WithCookieTemplate(func() http.Cookie {
			return http.Cookie{
				Name:     SessionCookieName,
				Path:     "/",
				Secure:   false,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode}
		}),
	}
	sessionStore, err := NewSessionStore(append(sessionStoreOptions, options...)...)
	require.NoError(t, err)
	return sessionStore
}

func setupEchoContext() echo.Context {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c
}

func TestCookie(t *testing.T) {
	sessionStore := setupSessionStore(t)
	session, err := sessionStore.sessionMaker.NewSession()
	require.NoError(t, err)

	cookie, err := sessionStore.cookie(session)
	require.NoError(t, err)
	assert.Equal(t, SessionCookieName, cookie.Name)
	assert.Equal(t, session.ID, cookie.Value)
}

func TestCookieWithSigning(t *testing.T) {
	hashKey := securecookie.GenerateRandomKey(32)
	encodingKey := securecookie.GenerateRandomKey(32)
	sessionStore := setupSessionStore(t, WithCookieHandler(securecookie.New(hashKey, encodingKey)))
	assert.NotNil(t, sessionStore.cookieHandler)
	session, err := sessionStore.sessionMaker.NewSession()
	require.NoError(t, err)

	cookie, err := sessionStore.cookie(session)
	require.NoError(t, err)
	assert.Equal(t, SessionCookieName, cookie.Name)
	assert.NotEqual(t, session.ID, cookie.Value)

	// Decode the encrypted value
	cookieHandler := securecookie.New(hashKey, encodingKey)
	var decoded string = ""
	err = cookieHandler.Decode(SessionCookieName, cookie.Value, &decoded)
	require.NoError(t, err)
	assert.Equal(t, session.ID, decoded)
}

func TestGetSessionIDFromCookie(t *testing.T) {
	sessionStore := setupSessionStore(t)
	session, err := sessionStore.sessionMaker.NewSession()
	require.NoError(t, err)
	cookie, err := sessionStore.cookie(session)
	require.NoError(t, err)

	c := setupEchoContext()
	c.Request().AddCookie(&cookie)

	sessionID, err := sessionStore.getSessionIDFromCookie(c)
	require.NoError(t, err)
	assert.Equal(t, session.ID, sessionID)
}

func TestGetSessionIDFromCookieWithSigning(t *testing.T) {
	hashKey := securecookie.GenerateRandomKey(32)
	encodingKey := securecookie.GenerateRandomKey(32)
	sessionStore := setupSessionStore(t, WithCookieHandler(securecookie.New(hashKey, encodingKey)))
	session, err := sessionStore.sessionMaker.NewSession()
	require.NoError(t, err)
	cookie, err := sessionStore.cookie(session)
	require.NoError(t, err)

	c := setupEchoContext()
	c.Request().AddCookie(&cookie)

	sessionID, err := sessionStore.getSessionIDFromCookie(c)
	require.NoError(t, err)
	assert.Equal(t, session.ID, sessionID)
}

func TestGetSessionIDFromCookieCannotTamperWithSigning(t *testing.T) {
	hashKey := securecookie.GenerateRandomKey(32)
	encodingKey := securecookie.GenerateRandomKey(32)
	sessionStore := setupSessionStore(t, WithCookieHandler(securecookie.New(hashKey, encodingKey)))
	session, err := sessionStore.sessionMaker.NewSession()
	require.NoError(t, err)
	cookie, err := sessionStore.cookie(session)
	require.NoError(t, err)
	cookie.Value = "fake-session-id"

	c := setupEchoContext()
	c.Request().AddCookie(&cookie)

	sessionID, err := sessionStore.getSessionIDFromCookie(c)
	require.NoError(t, err)
	assert.Equal(t, "", sessionID)
}

func TestGetSessionIDFromCookieNoCookie(t *testing.T) {
	sessionStore := setupSessionStore(t)

	c := setupEchoContext()

	sessionID, err := sessionStore.getSessionIDFromCookie(c)
	require.NoError(t, err)
	assert.Equal(t, "", sessionID)
}
