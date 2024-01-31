package login

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zitadel/oidc/v2/pkg/oidc"
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
	e.Use(middleware.Recover(), middleware.Logger())
	loginServer.RegisterHandlers(e)
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

	loginServerListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	loginServerPort := loginServerListener.Addr().(*net.TCPAddr).Port
	defer loginServerListener.Close()
	kcAuthServer := testAuthServer{
		Authorized:   true,
		RefreshToken: "refresh-token-value",
		ClientID:     "renku",
		CallbackURI:  fmt.Sprintf("http://127.0.0.1:%d/callback", loginServerPort),
		DefaultProvider: true,
		IssuedTokens: []string{},
	}
	kcAuthServerCli := testAuthServer{
		Authorized:   true,
		RefreshToken: "refresh-token-value",
		ClientID:     "renkucli",
		CallbackURI:  fmt.Sprintf("http://127.0.0.1:%d/callback", loginServerPort),
		DefaultProvider: false,
		IssuedTokens: []string{},
	}
	kcAuthServer.Start()
	kcAuthServerCli.Start()
	defer kcAuthServer.Server().Close()
	defer kcAuthServerCli.Server().Close()
	testConfig, err := getTestConfig(loginServerPort, kcAuthServer, kcAuthServerCli)
	require.NoError(t, err)

	api, err := NewLoginServer(WithConfig(testConfig))
	require.NoError(t, err)
	apiServer, err := startTestServer(api, loginServerListener)
	require.NoError(t, err)
	defer apiServer.Close()
	client := *http.DefaultClient
	jar, err := cookiejar.New(&cookiejar.Options{})
	require.NoError(t, err)
	client.Jar = jar
	testServerURL, err := url.Parse(strings.TrimRight(
		fmt.Sprintf("http://127.0.0.1:%d%s", loginServerPort, testConfig.EndpointsBasePath),
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
	assert.Equal(t, models.SessionCookieName, sessionCookie.Name)
	session, err := api.sessionStore.GetSession(context.Background(), sessionCookie.Value)
	require.NoError(t, err)
	assert.Len(t, session.TokenIDs, 1)
	assert.Equal(t, 0, session.ProviderIDs.Len())
	assert.Equal(t, res.Request.URL.String(), testConfig.RenkuBaseURL.String())

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
		CallbackURI:  fmt.Sprintf("http://127.0.0.1:%d/callback", loginServerPort),
		DefaultProvider: true,
		IssuedTokens: []string{},
	}
	kcAuthServer1.Start()
	defer kcAuthServer1.Server().Close()
	kcAuthServer2 := testAuthServer{
		Authorized:   true,
		RefreshToken: "refresh-token-value",
		ClientID:     "renkucli",
		CallbackURI:  fmt.Sprintf("http://127.0.0.1:%d/callback", loginServerPort),
		DefaultProvider: true,
		IssuedTokens: []string{},
	}
	kcAuthServer2.Start()
	defer kcAuthServer2.Server().Close()
	testConfig, err := getTestConfig(loginServerPort, kcAuthServer1, kcAuthServer2)
	require.NoError(t, err)

	api, err := NewLoginServer(WithConfig(testConfig))
	apiServer, err := startTestServer(api, loginServerListener)
	require.NoError(t, err)
	defer apiServer.Close()
	client := *http.DefaultClient
	jar, err := cookiejar.New(&cookiejar.Options{})
	require.NoError(t, err)
	client.Jar = jar

	require.NoError(t, err)
	testServerURL, err := url.Parse(strings.TrimRight(
		fmt.Sprintf("http://127.0.0.1:%d%s", loginServerPort, testConfig.EndpointsBasePath),
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
	assert.Equal(t, models.SessionCookieName, sessionCookie.Name)
	session, err := api.sessionStore.GetSession(context.Background(), sessionCookie.Value)
	require.NoError(t, err)
	assert.Len(t, session.TokenIDs, 2)
	assert.Equal(t, 0, session.ProviderIDs.Len())
	assert.Equal(t, res.Request.URL.String(), testConfig.RenkuBaseURL.String())

	req, err = http.NewRequest(http.MethodGet, testServerURL.JoinPath("/logout").String(), nil)
	require.NoError(t, err)
	res, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	session, err = api.sessionStore.GetSession(context.Background(), sessionCookie.Value)
	assert.ErrorIs(t, err, gwerrors.ErrSessionNotFound)
}

func TestLoginCLI(t *testing.T) {
	var err error

	loginServerListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	loginServerPort := loginServerListener.Addr().(*net.TCPAddr).Port
	defer loginServerListener.Close()

	gitlabAuth := testAuthServer{
		Authorized:   true,
		RefreshToken: "refresh-token-value",
		ClientID:     "gitlab",
		CallbackURI:  fmt.Sprintf("http://127.0.0.1:%d/callback", loginServerPort),
		DefaultProvider: true,
		IssuedTokens: []string{},
	}
	gitlabAuth.Start()
	defer gitlabAuth.Server().Close()
	kcAuthServer := testAuthServer{
		Authorized:   true,
		RefreshToken: "refresh-token-value",
		ClientID:     "renkucli",
		CallbackURI:  fmt.Sprintf("http://127.0.0.1:%d/callback", loginServerPort),
		DefaultProvider: false,
		IssuedTokens: []string{},
	}
	kcAuthServer.Start()
	defer kcAuthServer.Server().Close()
	testConfig, err := getTestConfig(loginServerPort, gitlabAuth, kcAuthServer)
	require.NoError(t, err)

	api, err := NewLoginServer(WithConfig(testConfig))
	apiServer, err := startTestServer(api, loginServerListener)
	require.NoError(t, err)
	defer apiServer.Close()
	cliJar, err := cookiejar.New(nil)
	require.NoError(t, err)
	cliClient := http.Client{Jar: cliJar}
	userJar, err := cookiejar.New(nil)
	require.NoError(t, err)
	userClient := http.Client{Jar: userJar}

	// Health check works
	require.NoError(t, err)
	testServerURL, err := url.Parse(strings.TrimRight(
		fmt.Sprintf("http://127.0.0.1:%d%s", loginServerPort, testConfig.EndpointsBasePath),
		"/",
	))
	require.NoError(t, err)
	assert.Len(t, cliClient.Jar.Cookies(testServerURL), 0)
	res, err := cliClient.Get(testServerURL.JoinPath("/health").String())
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	// The CLI asks for a device login URL
	req, err := http.NewRequest(http.MethodPost, testServerURL.JoinPath("/device/login").String(), nil)
	require.NoError(t, err)
	res, err = cliClient.Do(req)
	require.NoError(t, err)
	var credentials oidc.DeviceAuthorizationResponse
	var resContent []byte
	errJSON := json.NewDecoder(res.Body).Decode(&credentials)
	if errJSON != nil {
		resContent, err = httputil.DumpResponse(res, true)
		require.NoError(t, err)
	}
	require.Equalf(
		t,
		http.StatusOK,
		res.StatusCode,
		"The status code %d != %d, response dump: %s, json dump: %+v",
		http.StatusOK,
		res.StatusCode,
		resContent,
		credentials,
	)
	assert.Len(t, cliClient.Jar.Cookies(testServerURL), 1)
	sessionCookie := cliClient.Jar.Cookies(testServerURL)[0]
	verificationURL, err := url.Parse(credentials.VerificationURI)
	require.NoError(t, err)
	assert.Equal(t, sessionCookie.Value, verificationURL.Query().Get(cliLoginSessionIDQueryParam))
	
	// The user visits the login URL that the CLI displayed
	req, err = http.NewRequest(http.MethodGet, verificationURL.String(), nil)
	require.NoError(t, err)
	res, err = userClient.Do(req)
	require.NoError(t, err)
	resContent, err = httputil.DumpResponse(res, true)
	require.NoError(t, err)
	assert.Equalf(t, http.StatusOK, res.StatusCode, "response %s, path: %s", string(resContent), res.Request.URL.String())
	session, err := api.sessionStore.GetSession(context.Background(), sessionCookie.Value)
	require.NoError(t, err)
	token, err := session.GetAccessToken(context.Background(), "gitlab")
	require.NoError(t, err)
	assert.Contains(t, gitlabAuth.IssuedTokens, token.Value)

	// The CLI checks if the device flow has completed
	req, err = http.NewRequest(http.MethodPost, testServerURL.JoinPath("/device/token").String(), nil)
	require.NoError(t, err)
	res, err = cliClient.Do(req)
	require.NoError(t, err)
	resContent, err = httputil.DumpResponse(res, true)
	require.NoError(t, err)
	var tokenResponse oidc.AccessTokenResponse
	errJSON = json.NewDecoder(res.Body).Decode(&tokenResponse)
	if errJSON != nil {
		resContent, err = httputil.DumpResponse(res, true)
		require.NoError(t, err)
	}
	assert.Equalf(t, http.StatusOK, res.StatusCode, "response dump %s, json dump %+v, path: %s", string(resContent), tokenResponse, res.Request.URL.String())
	// The CLI is not supposed to get the credentials
	require.NoError(t, errJSON)
	assert.Equal(t, "redacted", tokenResponse.AccessToken)
	assert.Equal(t, "redacted", tokenResponse.RefreshToken)
	assert.Equal(t, "redacted", tokenResponse.IDToken)

	// After the CLI has checked and received the credentials it 
}

