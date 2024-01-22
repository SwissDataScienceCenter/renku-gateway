package login

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestProviderConfig(authServers ...testAuthServer) (map[string]config.OIDCClient, error) {
	output := map[string]config.OIDCClient{}
	for _, server := range authServers {
		id, err := uuid.NewUUID()
		if err != nil {
			return map[string]config.OIDCClient{}, err
		}
		output[id.String()] = server.ProviderConfig()
	}
	return output, nil
}

func getTestConfig(loginServerPort int, authServers ...testAuthServer) (config.LoginConfig, error) {
	providers, err := getTestProviderConfig(authServers...)
	if err != nil {
		return config.LoginConfig{}, err
	}
	config := config.LoginConfig{
		DefaultAppRedirectURL: fmt.Sprintf("http://localhost:%d/health", loginServerPort),
		TokenEncryption: config.TokenEncryptionConfig{
			Enabled:   true,
			SecretKey: "1b195c6329ba7df1c1adf6975c71910d",
		},
		Providers: providers,
	}
	return config, nil
}

func startTestServer(loginServer *LoginServer, listener net.Listener) (*httptest.Server, error) {
	e := echo.New()
	loginServer.RegisterHandlers(e)
	server := httptest.NewUnstartedServer(e)
	server.Listener = listener
	server.Start()
	return server, nil
}

func TestGetLogin(t *testing.T) {
	var err error

	loginServerListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	loginServerPort := loginServerListener.Addr().(*net.TCPAddr).Port
	defer loginServerListener.Close()
	kcAuthServer := testAuthServer{
		Authorized:   true,
		RefreshToken: "refresh-token-value",
		ClientID:     "renku",
		CallbackURI:  fmt.Sprintf("http://localhost:%d/callback", loginServerPort),
	}
	kcAuthServer.Start()
	defer kcAuthServer.Server().Close()
	config, err := getTestConfig(loginServerPort, kcAuthServer)
	require.NoError(t, err)

	api, err := NewLoginServer(WithConfig(config))
	require.NoError(t, err)
	apiServer, err := startTestServer(api, loginServerListener)
	require.NoError(t, err)
	defer apiServer.Close()
	client := *http.DefaultClient
	jar, err := cookiejar.New(&cookiejar.Options{})
	require.NoError(t, err)
	client.Jar = jar
	testServerURL, err := url.Parse(strings.TrimRight(
		fmt.Sprintf("http://localhost:%d%s", loginServerPort, config.EndpointsBasePath),
		"/",
	))
	require.NoError(t, err)
	assert.Len(t, client.Jar.Cookies(testServerURL), 0)
	res, err := client.Get(testServerURL.JoinPath("/health").String())
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	req, err := http.NewRequest(http.MethodGet, testServerURL.JoinPath("/login").String(), nil)
	require.NoError(t, err)
	res, err = client.Do(req)
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
	assert.Equal(t, api.sessionHandler.Cookie(&models.Session{TTLSeconds: 3600, CreatedAt: time.Now().UTC()}).Name, sessionCookie.Name)
	session, err := api.sessionStore.GetSession(context.Background(), sessionCookie.Value)
	require.NoError(t, err)
	assert.Len(t, session.TokenIDs, 1)
	assert.Equal(t, 0, session.ProviderIDs.Len())
	assert.Equal(t, res.Request.URL.String(), config.DefaultAppRedirectURL)

	req, err = http.NewRequest(http.MethodGet, testServerURL.JoinPath("/logout").String(), nil)
	require.NoError(t, err)
	res, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	session, err = api.sessionStore.GetSession(context.Background(), sessionCookie.Value)
	assert.ErrorIs(t, err, gwerrors.ErrSessionNotFound)
}

func TestGetLogin2Steps(t *testing.T) {
	var err error

	loginServerListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	loginServerPort := loginServerListener.Addr().(*net.TCPAddr).Port
	defer loginServerListener.Close()

	kcAuthServer1 := testAuthServer{
		Authorized:   true,
		RefreshToken: "refresh-token-value",
		ClientID:     "renku1",
		CallbackURI:  fmt.Sprintf("http://localhost:%d/callback", loginServerPort),
	}
	kcAuthServer1.Start()
	defer kcAuthServer1.Server().Close()
	kcAuthServer2 := testAuthServer{
		Authorized:   true,
		RefreshToken: "refresh-token-value",
		ClientID:     "renku2",
		CallbackURI:  fmt.Sprintf("http://localhost:%d/callback", loginServerPort),
	}
	kcAuthServer2.Start()
	defer kcAuthServer2.Server().Close()
	config, err := getTestConfig(loginServerPort, kcAuthServer1, kcAuthServer2)
	require.NoError(t, err)

	api, err := NewLoginServer(WithConfig(config))
	apiServer, err := startTestServer(api, loginServerListener)
	require.NoError(t, err)
	defer apiServer.Close()
	client := *http.DefaultClient
	jar, err := cookiejar.New(&cookiejar.Options{})
	require.NoError(t, err)
	client.Jar = jar

	require.NoError(t, err)
	testServerURL, err := url.Parse(strings.TrimRight(
		fmt.Sprintf("http://localhost:%d%s", loginServerPort, config.EndpointsBasePath),
		"/",
	))
	require.NoError(t, err)
	assert.Len(t, client.Jar.Cookies(testServerURL), 0)
	res, err := client.Get(testServerURL.JoinPath("/health").String())
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	req, err := http.NewRequest(http.MethodGet, testServerURL.JoinPath("/login").String(), nil)
	require.NoError(t, err)
	res, err = client.Do(req)
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
	assert.Equal(t, api.sessionHandler.Cookie(&models.Session{TTLSeconds: 3600, CreatedAt: time.Now().UTC()}).Name, sessionCookie.Name)
	session, err := api.sessionStore.GetSession(context.Background(), sessionCookie.Value)
	require.NoError(t, err)
	assert.Len(t, session.TokenIDs, 2)
	assert.Equal(t, 0, session.ProviderIDs.Len())
	assert.Equal(t, res.Request.URL.String(), config.DefaultAppRedirectURL)

	req, err = http.NewRequest(http.MethodGet, testServerURL.JoinPath("/logout").String(), nil)
	require.NoError(t, err)
	res, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	session, err = api.sessionStore.GetSession(context.Background(), sessionCookie.Value)
	assert.ErrorIs(t, err, gwerrors.ErrSessionNotFound)
}
