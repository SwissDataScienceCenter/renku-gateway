package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/commonconfig"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestProviderConfig(authServers ...testAuthServer) (string, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp("", id.String())

	if err != nil {
		return "", err
	}
	defer f.Close()

	for _, server := range authServers {
		_, err := f.Write([]byte(server.ProviderConfig() + "\n"))
		if err != nil {
			return "", err
		}
	}

	return f.Name(), nil
}

func getTestConfig(
	providersConfigFile string,
	defaultProviderIDs []string,
	loginServerPort int,
) (LoginServerConfig, error) {
	config := LoginServerConfig{
		Server: ServerConfig{
			BasePath: "/api/auth",
			Port:     loginServerPort,
		},
		DefaultProviderIDs:    defaultProviderIDs,
		DefaultAppRedirectURL: fmt.Sprintf("http://localhost:%d/api/auth/health", loginServerPort),
		CallbackURL:           fmt.Sprintf("http://localhost:%d/api/auth/callback", loginServerPort),
		SessionPersistence: SessionPersistenceConfig{
			Type: "redis-mock",
		},
		TokenEncryption: TokenEncryptionConfig{
			Enabled:   true,
			SecretKey: "1b195c6329ba7df1c1adf6975c71910d",
		},
		ProviderConfigFile:     providersConfigFile,
		sessionCookieNotSecure: true,
	}
	return config, nil
}

func startTestServer(loginServer *LoginServer, listener net.Listener) (*httptest.Server, error) {
	server := httptest.NewUnstartedServer(loginServer.echo)
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
		CallbackURI:  fmt.Sprintf("http://localhost:%d/api/auth/callback", loginServerPort),
	}
	kcAuthServer.Start()
	defer kcAuthServer.Server().Close()
	providersConfigFile, err := getTestProviderConfig(kcAuthServer)
	require.NoError(t, err)
	defer os.Remove(providersConfigFile)
	config, err := getTestConfig(providersConfigFile, []string{"renku"}, loginServerPort)
	require.NoError(t, err)

	api, err := NewLoginServer(&config)
	require.NoError(t, err)
	apiServer, err := startTestServer(api, loginServerListener)
	require.NoError(t, err)
	defer apiServer.Close()
	client := *http.DefaultClient
	jar, err := cookiejar.New(&cookiejar.Options{})
	require.NoError(t, err)
	client.Jar = jar
	testServerURL, err := url.Parse(strings.TrimRight(
		fmt.Sprintf("http://localhost:%d%s", config.Server.Port, config.Server.BasePath),
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
	assert.Equal(t, commonconfig.SessionCookieName, sessionCookie.Name)
	session, err := api.sessionStore.GetSession(context.Background(), sessionCookie.Value)
	require.NoError(t, err)
	assert.Len(t, session.TokenIDs, 1)
	assert.Len(t, session.LoginWithProviders, 0)
	assert.Equal(t, res.Request.URL.String(), config.DefaultAppRedirectURL)

	req, err = http.NewRequest(http.MethodGet, testServerURL.JoinPath("/logout").String(), nil)
	require.NoError(t, err)
	res, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	session, err = api.sessionStore.GetSession(context.Background(), sessionCookie.Value)
	require.NoError(t, err)
	assert.Equal(t, models.Session{}, session)
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
		CallbackURI:  fmt.Sprintf("http://localhost:%d/api/auth/callback", loginServerPort),
	}
	kcAuthServer1.Start()
	defer kcAuthServer1.Server().Close()
	kcAuthServer2 := testAuthServer{
		Authorized:   true,
		RefreshToken: "refresh-token-value",
		ClientID:     "renku2",
		CallbackURI:  fmt.Sprintf("http://localhost:%d/api/auth/callback", loginServerPort),
	}
	kcAuthServer2.Start()
	defer kcAuthServer2.Server().Close()
	providersConfigFile, err := getTestProviderConfig(kcAuthServer1, kcAuthServer2)
	require.NoError(t, err)
	defer os.Remove(providersConfigFile)
	config, err := getTestConfig(providersConfigFile, []string{"renku1", "renku2"}, loginServerPort)
	require.NoError(t, err)

	api, err := NewLoginServer(&config)
	apiServer, err := startTestServer(api, loginServerListener)
	require.NoError(t, err)
	defer apiServer.Close()
	client := *http.DefaultClient
	jar, err := cookiejar.New(&cookiejar.Options{})
	require.NoError(t, err)
	client.Jar = jar

	require.NoError(t, err)
	testServerURL, err := url.Parse(strings.TrimRight(
		fmt.Sprintf("http://localhost:%d%s", config.Server.Port, config.Server.BasePath),
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
	assert.Equal(t, commonconfig.SessionCookieName, sessionCookie.Name)
	session, err := api.sessionStore.GetSession(context.Background(), sessionCookie.Value)
	require.NoError(t, err)
	assert.Len(t, session.TokenIDs, 2)
	assert.Len(t, session.LoginWithProviders, 0)
	assert.Equal(t, res.Request.URL.String(), config.DefaultAppRedirectURL)

	req, err = http.NewRequest(http.MethodGet, testServerURL.JoinPath("/logout").String(), nil)
	require.NoError(t, err)
	res, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	session, err = api.sessionStore.GetSession(context.Background(), sessionCookie.Value)
	require.NoError(t, err)
	assert.Equal(t, models.Session{}, session)
}
