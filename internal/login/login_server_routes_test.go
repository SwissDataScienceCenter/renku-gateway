package login

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/authentication"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/db"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/tokenstore"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/views"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestProviderConfig(authServers ...testAuthServer) (map[string]config.OIDCClient, error) {
	output := map[string]config.OIDCClient{}
	for _, server := range authServers {
		output[server.ClientID] = server.ProviderConfig()
	}
	return output, nil
}

func getTestConfig(loginServerPort int, authServers ...testAuthServer) (config.LoginConfig, error) {
	providers, err := getTestProviderConfig(authServers...)
	if err != nil {
		return config.LoginConfig{}, err
	}
	renkuBaseURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", loginServerPort))
	if err != nil {
		return config.LoginConfig{}, err
	}
	testConfig := config.LoginConfig{
		RenkuBaseURL: renkuBaseURL,
		TokenEncryption: config.TokenEncryptionConfig{
			Enabled:   true,
			SecretKey: "1b195c6329ba7df1c1adf6975c71910d",
		},
		Providers: providers,
	}
	return testConfig, nil
}

func startTestServer(loginServer *LoginServer, listener net.Listener) (*httptest.Server, error) {
	e := echo.New()
	e.Pre(middleware.RequestID())
	e.Use(middleware.Recover(), middleware.Logger())
	tr, err := views.NewTemplateRenderer()
	if err != nil {
		return nil, err
	}
	tr.Register(e)
	loginServer.RegisterHandlers(e, loginServer.sessions.Middleware())
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "You have reached the Renku home page")
	})
	server := httptest.NewUnstartedServer(e)
	server.Listener = listener
	server.Start()
	return server, nil
}

func TestGetLogin(t *testing.T) {
	var err error

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	loginServerListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	loginServerPort := loginServerListener.Addr().(*net.TCPAddr).Port
	defer loginServerListener.Close()
	kcAuthServer := testAuthServer{
		Authorized:      true,
		RefreshToken:    "refresh-token-value",
		ClientID:        "renku",
		CallbackURI:     fmt.Sprintf("http://127.0.0.1:%d/callback", loginServerPort),
		DefaultProvider: true,
		IssuedTokens:    []string{},
	}
	kcAuthServer.Start()
	defer kcAuthServer.Server().Close()
	testConfig, err := getTestConfig(loginServerPort, kcAuthServer)
	require.NoError(t, err)

	dbAdapter, err := db.NewRedisAdapter(db.WithRedisConfig(config.RedisConfig{
		Type: config.DBTypeRedisMock,
	}))
	require.NoError(t, err)
	tokenStore, err := tokenstore.NewTokenStore(
		tokenstore.WithExpiryMargin(time.Duration(3)*time.Minute),
		tokenstore.WithConfig(testConfig),
		tokenstore.WithTokenRepository(dbAdapter),
	)
	require.NoError(t, err)
	authenticator, err := authentication.NewAuthenticator()
	require.NoError(t, err)
	sessionStore, err := sessions.NewSessionStore(
		sessions.WithAuthenticator(authenticator),
		sessions.WithSessionRepository(dbAdapter),
		sessions.WithTokenStore(tokenStore),
		sessions.WithConfig(config.SessionConfig{}),
		sessions.WithCookieTemplate(func() http.Cookie {
			return http.Cookie{
				Name:     sessions.SessionCookieName,
				Path:     "/",
				Secure:   false,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode}
		}),
	)
	require.NoError(t, err)
	api, err := NewLoginServer(
		WithConfig(testConfig),
		WithSessionStore(sessionStore),
		WithTokenStore(tokenStore),
	)
	require.NoError(t, err)
	apiServer, err := startTestServer(api, loginServerListener)
	require.NoError(t, err)
	defer apiServer.Close()
	client := *http.DefaultClient
	jar, err := cookiejar.New(&cookiejar.Options{})
	require.NoError(t, err)
	client.Jar = jar
	testServerURL, err := url.Parse(strings.TrimRight(
		fmt.Sprintf("http://127.0.0.1:%d%s", loginServerPort, testConfig.LoginRoutesBasePath),
		"/",
	))
	require.NoError(t, err)
	assert.Len(t, client.Jar.Cookies(testServerURL), 0)

	loginURL := testServerURL.JoinPath("/login")
	v := url.Values{}
	v.Add("provider_id", "renku")
	loginURL.RawQuery = v.Encode()
	req, err := http.NewRequest(http.MethodGet, loginURL.String(), nil)
	require.NoError(t, err)
	res, err := client.Do(req)
	require.NoError(t, err)
	resContent, err := httputil.DumpResponse(res, true)
	require.NoError(t, err)
	require.Equalf(
		t,
		http.StatusOK,
		res.StatusCode,
		"Response code is not %d but %d, response dump is: %s",
		http.StatusOK,
		res.StatusCode,
		resContent,
	)
	assert.Len(t, client.Jar.Cookies(testServerURL), 1)

	sessionCookie := client.Jar.Cookies(testServerURL)[0]
	assert.Equal(t, sessions.SessionCookieName, sessionCookie.Name)
	session, err := dbAdapter.GetSession(context.Background(), sessionCookie.Value)
	require.NoError(t, err)
	assert.Len(t, session.TokenIDs, 1)
	assert.Len(t, session.LoginSequence, 0)
	assert.Equal(t, "", session.LoginState)
	assert.Equal(t, res.Request.URL.String(), testConfig.RenkuBaseURL.String())

	req, err = http.NewRequest(http.MethodGet, testServerURL.JoinPath("/logout").String(), nil)
	require.NoError(t, err)
	res, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	session, err = dbAdapter.GetSession(context.Background(), sessionCookie.Value)
	assert.ErrorIs(t, err, gwerrors.ErrSessionNotFound)
}

func TestGetLogin2Steps(t *testing.T) {
	var err error

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	loginServerListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	loginServerPort := loginServerListener.Addr().(*net.TCPAddr).Port
	defer loginServerListener.Close()

	kcAuthServer1 := testAuthServer{
		Authorized:      true,
		RefreshToken:    "refresh-token-value",
		ClientID:        "renku",
		CallbackURI:     fmt.Sprintf("http://127.0.0.1:%d/callback", loginServerPort),
		DefaultProvider: true,
		IssuedTokens:    []string{},
	}
	kcAuthServer1.Start()
	defer kcAuthServer1.Server().Close()
	kcAuthServer2 := testAuthServer{
		Authorized:      true,
		RefreshToken:    "refresh-token-value",
		ClientID:        "gitlab",
		CallbackURI:     fmt.Sprintf("http://127.0.0.1:%d/callback", loginServerPort),
		DefaultProvider: true,
		IssuedTokens:    []string{},
	}
	kcAuthServer2.Start()
	defer kcAuthServer2.Server().Close()
	testConfig, err := getTestConfig(loginServerPort, kcAuthServer1, kcAuthServer2)
	require.NoError(t, err)

	dbAdapter, err := db.NewRedisAdapter(db.WithRedisConfig(config.RedisConfig{
		Type: config.DBTypeRedisMock,
	}))
	require.NoError(t, err)
	tokenStore, err := tokenstore.NewTokenStore(
		tokenstore.WithExpiryMargin(time.Duration(3)*time.Minute),
		tokenstore.WithConfig(testConfig),
		tokenstore.WithTokenRepository(dbAdapter),
	)
	require.NoError(t, err)
	authenticator, err := authentication.NewAuthenticator()
	require.NoError(t, err)
	sessionStore, err := sessions.NewSessionStore(
		sessions.WithAuthenticator(authenticator),
		sessions.WithSessionRepository(dbAdapter),
		sessions.WithTokenStore(tokenStore),
		sessions.WithConfig(config.SessionConfig{}),
		sessions.WithCookieTemplate(func() http.Cookie {
			return http.Cookie{
				Name:     sessions.SessionCookieName,
				Path:     "/",
				Secure:   false,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode}
		}),
	)
	require.NoError(t, err)
	api, err := NewLoginServer(
		WithConfig(testConfig),
		WithSessionStore(sessionStore),
		WithTokenStore(tokenStore),
	)
	require.NoError(t, err)
	apiServer, err := startTestServer(api, loginServerListener)
	require.NoError(t, err)
	defer apiServer.Close()
	client := *http.DefaultClient
	jar, err := cookiejar.New(&cookiejar.Options{})
	require.NoError(t, err)
	client.Jar = jar

	require.NoError(t, err)
	testServerURL, err := url.Parse(strings.TrimRight(
		fmt.Sprintf("http://127.0.0.1:%d%s", loginServerPort, testConfig.LoginRoutesBasePath),
		"/",
	))
	require.NoError(t, err)
	assert.Len(t, client.Jar.Cookies(testServerURL), 0)

	req, err := http.NewRequest(http.MethodGet, testServerURL.JoinPath("/login").String(), nil)
	require.NoError(t, err)
	res, err := client.Do(req)
	require.NoError(t, err)
	resContent, err := httputil.DumpResponse(res, true)
	require.NoError(t, err)
	assert.Equalf(
		t,
		http.StatusOK,
		res.StatusCode,
		"The status code %d != %d, response dump: %s",
		http.StatusOK,
		res.StatusCode,
		resContent,
	)
	assert.Len(t, client.Jar.Cookies(testServerURL), 1)

	sessionCookie := client.Jar.Cookies(testServerURL)[0]
	assert.Equal(t, sessions.SessionCookieName, sessionCookie.Name)
	session, err := dbAdapter.GetSession(context.Background(), sessionCookie.Value)
	require.NoError(t, err)
	assert.Len(t, session.TokenIDs, 2)
	assert.Len(t, session.LoginSequence, 0)
	assert.Equal(t, "", session.LoginState)
	assert.Equal(t, res.Request.URL.String(), testConfig.RenkuBaseURL.String())

	req, err = http.NewRequest(http.MethodGet, testServerURL.JoinPath("/logout").String(), nil)
	require.NoError(t, err)
	res, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	session, err = dbAdapter.GetSession(context.Background(), sessionCookie.Value)
	assert.ErrorIs(t, err, gwerrors.ErrSessionNotFound)
}
