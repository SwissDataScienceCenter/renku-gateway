package revproxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/db"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const serverIDHeader string = "Server-ID"

func sessionExpired() models.SessionOption {
	return func(s *models.Session) error {
		s.TTLSeconds = 60
		s.CreatedAt = time.Now().UTC().Add(time.Hour * -5)
		return nil
	}
}

func withTokenIDs(ids ...string) models.SessionOption {
	return func(s *models.Session) error {
		s.TokenIDs = ids
		return nil
	}
}

func sessionID(id string) models.SessionOption {
	return func(s *models.Session) error {
		s.ID = id
		return nil
	}
}

func newTestSesssion(options ...models.SessionOption) models.Session {
	session, err := models.NewSession(options...)
	if err != nil {
		log.Fatal(err)
	}
	return session
}

type tokenOption func(*models.AuthToken)

func tokenID(id string) tokenOption {
	return func(t *models.AuthToken) {
		t.ID = id
	}
}

func tokenPlainValue(val string) tokenOption {
	return func(t *models.AuthToken) {
		t.Value = val
	}
}

type customClaims struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func tokenJWTValue(claims customClaims) tokenOption {
	return func(t *models.AuthToken) {
		if claims.ExpiresAt == nil {
			claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour * 5))
		}
		token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
		signed, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
		if err != nil {
			log.Fatalln(err)
		}
		t.Value = signed
	}
}

func tokenExpired(val string) tokenOption {
	return func(t *models.AuthToken) {
		t.ExpiresAt = time.Now().UTC().Add(time.Hour * -5)
	}
}

func tokenProviderID(id string) tokenOption {
	return func(t *models.AuthToken) {
		t.ProviderID = id
	}
}

func newTestToken(tokenType models.OauthTokenType, options ...tokenOption) models.AuthToken {
	token := models.AuthToken{
		Type:      tokenType,
		ID:        "tokenID",
		Value:     "tokenValue",
		ExpiresAt: time.Now().UTC().Add(time.Hour * 5),
	}
	for _, opt := range options {
		opt(&token)
	}
	return token
}

type testRequestTracker chan *http.Request

func (t testRequestTracker) getAllRequests() []*http.Request {
	close(t)
	reqs := []*http.Request{}
	for req := range t {
		reqs = append(reqs, req)
	}
	return reqs
}

func setupTestUpstream(ID string, requestTracker chan<- *http.Request) (*httptest.Server, *url.URL) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(w, r.Body)
		for k := range r.Header {
			v := r.Header.Get(k)
			w.Header().Set(k, v)
		}
		r.Header.Set(serverIDHeader, ID)
		requestTracker <- r
		w.WriteHeader(http.StatusOK)
	}))
	url, err := url.Parse(srv.URL)
	if err != nil {
		log.Fatal(err)
	}
	return srv, url
}

func setupTestAuthServer(
	ID string,
	responseHeaders map[string]string,
	responseStatus int,
	requestTracker chan<- *http.Request,
) (*httptest.Server, *url.URL) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range responseHeaders {
			w.Header().Set(k, v)
		}
		r.Header.Set(serverIDHeader, ID)
		requestTracker <- r
		w.WriteHeader(responseStatus)
	}))
	url, err := url.Parse(srv.URL)
	if err != nil {
		log.Fatal(err)
	}
	return srv, url
}

func setupTestRevproxy(rpConfig *config.RevproxyConfig, sessions *sessions.SessionStore) (*httptest.Server, *url.URL) {
	proxy, err := NewServer(WithConfig(*rpConfig), WithSessionStore(sessions))
	if err != nil {
		log.Fatal(err)
	}
	e := echo.New()
	e.Pre(middleware.RemoveTrailingSlash(), UiServerPathRewrite())
	e.Use(middleware.Recover(), middleware.Logger())
	proxy.RegisterHandlers(e, sessions.Middleware())
	srv := httptest.NewServer(e)
	url, err := url.Parse(srv.URL)
	if err != nil {
		log.Fatal(err)
	}
	return srv, url
}

type TestResults struct {
	Path                     string
	VisitedServerIDs         []string
	ResponseHeaders          map[string]string
	Non200ResponseStatusCode int
	IgnoreErrors             bool
	UpstreamRequestHeaders   []map[string]string
}

type TestCase struct {
	Path           string
	QueryParams    map[string]string
	Tokens         []models.AuthToken
	Sessions       []models.Session
	ExternalGitlab bool
	Expected       TestResults
	RequestHeader  map[string]string
	RequestCookie  *http.Cookie
}

func ParametrizedRouteTest(scenario TestCase) func(*testing.T) {
	return func(t *testing.T) {
		// Setup and start
		requestTracker := make(testRequestTracker, 20)
		upstream, upstreamURL := setupTestUpstream("upstream", requestTracker)
		upstream2, upstreamURL2 := setupTestUpstream("upstream2", requestTracker)
		var (
			gitlab    *httptest.Server
			gitlabURL *url.URL
			err       error
		)
		rpConfig := config.RevproxyConfig{
			RenkuBaseURL: upstreamURL,
			RenkuServices: config.RenkuServicesConfig{
				Notebooks: upstreamURL,
				Core: config.CoreSvcConfig{
					ServiceNames: []string{upstreamURL.String(), upstreamURL.String(), upstreamURL2.String()},
					ServicePaths: []string{"/api/renku", "/api/renku/10", "/api/renku/9"},
					Sticky:       false,
				},
				KG:          upstreamURL,
				Webhook:     upstreamURL,
				DataService: upstreamURL,
				Keycloak:    upstreamURL,
				UIServer:    upstreamURL,
			},
		}
		dbAdapter := db.NewMockRedisAdapter()
		for _, token := range scenario.Tokens {
			switch token.Type {
			case models.AccessTokenType:
				err = dbAdapter.SetAccessToken(context.Background(), token)
			case models.RefreshTokenType:
				err = dbAdapter.SetRefreshToken(context.Background(), token)
			case models.IDTokenType:
				err = dbAdapter.SetIDToken(context.Background(), token)
			default:
				err = fmt.Errorf("unrecognized token type %s", token.Type)
			}
			require.NoError(t, err)
		}
		for _, session := range scenario.Sessions {
			err := dbAdapter.SetSession(context.Background(), session)
			require.NoError(t, err)
		}
		sh := models.NewSessionHandler(models.WithTokenStore(&dbAdapter), models.WithSessionStore(&dbAdapter))
		if scenario.ExternalGitlab {
			gitlab, gitlabURL = setupTestUpstream("gitlab", requestTracker)
			defer gitlab.Close()
			rpConfig.ExternalGitlabURL = gitlabURL
		}
		proxy, proxyURL := setupTestRevproxy(&rpConfig, &sh)
		defer upstream.Close()
		defer upstream2.Close()
		defer proxy.Close()

		// Make request through proxy
		reqURL := proxyURL.JoinPath(scenario.Path)
		reqURLQuery := reqURL.Query()
		for k, v := range scenario.QueryParams {
			reqURLQuery.Add(k, v)
		}
		reqURL.RawQuery = reqURLQuery.Encode()
		// Force ipv4 becaues Github actions seem to constantly switch to ipv6 and fail
		transport := http.DefaultTransport.(*http.Transport).Clone()
		var zeroDialer net.Dialer
		var httpClient = &http.Client{
			Timeout: 10 * time.Second,
		}
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return zeroDialer.DialContext(ctx, "tcp4", addr)
		}
		httpClient.Transport = transport
		testReq, err := http.NewRequest("GET", reqURL.String(), nil)
		require.NoError(t, err)
		for k, v := range scenario.RequestHeader {
			testReq.Header.Set(k, v)
		}
		if scenario.RequestCookie != nil {
			testReq.AddCookie(scenario.RequestCookie)
		}
		res, err := httpClient.Do(testReq)
		assert.NoError(t, err)
		reqs := requestTracker.getAllRequests()

		// Assert the request was routed as expected
		if !scenario.Expected.IgnoreErrors {
			assert.NoError(t, err)
		}
		if scenario.Expected.Non200ResponseStatusCode != 0 {
			resContent, err := httputil.DumpResponse(res, true)
			require.NoError(t, err)
			assert.Equalf(
				t,
				scenario.Expected.Non200ResponseStatusCode,
				res.StatusCode,
				"The status code is not as expected %d, it is %d, response body is %s",
				scenario.Expected.Non200ResponseStatusCode,
				res.StatusCode,
				string(resContent),
			)
		} else {
			resContent, err := httputil.DumpResponse(res, true)
			require.NoError(t, err)
			assert.Equalf(t, http.StatusOK, res.StatusCode, "The status code is not 200, it is %d, response body is %s", res.StatusCode, string(resContent))
		}
		assert.Len(t, reqs, len(scenario.Expected.VisitedServerIDs))
		if len(reqs) == len(scenario.Expected.VisitedServerIDs) {
			for ireq, req := range reqs {
				assert.Equal(t, scenario.Expected.VisitedServerIDs[ireq], req.Header.Get(serverIDHeader))
			}
		}
		for hdrKey, hdrValue := range scenario.Expected.ResponseHeaders {
			assert.Equal(t, hdrValue, res.Header.Get(hdrKey))
		}
		if scenario.Expected.Path != "" && len(reqs) > 0 {
			assert.Equal(t, scenario.Expected.Path, reqs[len(reqs)-1].URL.EscapedPath())
		}
		if len(scenario.QueryParams) > 0 && len(reqs) > 0 {
			assert.Equal(t, reqURLQuery.Encode(), reqs[len(reqs)-1].URL.RawQuery)
		}
		if scenario.Expected.UpstreamRequestHeaders != nil {
			require.Equal(t, len(reqs), len(scenario.Expected.UpstreamRequestHeaders))
		}
		for ireq, expectedHeaders := range scenario.Expected.UpstreamRequestHeaders {
			req := reqs[ireq]
			for k, v := range expectedHeaders {
				assert.Equal(t, v, req.Header.Get(k))
			}
		}
	}
}

func TestInternalSvcRoutes(t *testing.T) {
	testCases := []TestCase{
		{
			Path: "/api/notebooks/test/rejectedAuth",
			Expected: TestResults{
				VisitedServerIDs: []string{"upstream"},
				UpstreamRequestHeaders: []map[string]string{{
					"Authorization":            "",
					"Renku-Auth-Id-Token":      "",
					"Renku-Auth-Access-Token":  "",
					"Renku-Auth-Refresh-Token": "",
					"Renku-Auth-Anon-Id":       "sessionID",
				}},
			},
			Sessions: []models.Session{
				newTestSesssion(sessionID("sessionID")),
			},
			RequestCookie: &http.Cookie{Name: models.SessionCookieName, Value: "sessionID"},
		},
		{
			Path: "/api/notebooks/test/acceptedAuth",
			Expected: TestResults{
				Path:             "/notebooks/test/acceptedAuth",
				VisitedServerIDs: []string{"upstream"},
				UpstreamRequestHeaders: []map[string]string{{
					"Renku-Auth-Id-Token":      "idTokenValue",
					"Renku-Auth-Access-Token":  "accessTokenValue",
					"Renku-Auth-Refresh-Token": "refreshTokenValue",
					"Renku-Auth-Anon-Id":       "",
				}},
			},
			Tokens: []models.AuthToken{
				newTestToken(
					models.AccessTokenType,
					tokenID("accessTokenID"),
					tokenPlainValue("accessTokenValue"),
					tokenProviderID("renku"),
				),
				newTestToken(
					models.RefreshTokenType,
					tokenID("refreshTokenID"),
					tokenPlainValue("refreshTokenValue"),
					tokenProviderID("renku"),
				),
				newTestToken(
					models.IDTokenType,
					tokenID("idTokenID"),
					tokenPlainValue("idTokenValue"),
					tokenProviderID("renku"),
				),
			},
			Sessions: []models.Session{
				newTestSesssion(sessionID("sessionID"), withTokenIDs("accessTokenID", "refreshTokenID", "idTokenID")),
			},
			RequestCookie: &http.Cookie{Name: models.SessionCookieName, Value: "sessionID"},
		},
		{
			Path:     "/api/notebooks",
			Expected: TestResults{Path: "/notebooks", VisitedServerIDs: []string{"upstream"}},
		},
		{
			Path: "/api/projects/123456/graph/status/something/else",
			Expected: TestResults{
				Path:             "/projects/123456/events/status/something/else",
				VisitedServerIDs: []string{"upstream"},
				UpstreamRequestHeaders: []map[string]string{{
					echo.HeaderAuthorization: "",
				}},
			},
		},
		{
			Path: "/api/projects/123456/graph/status/something/else",
			Expected: TestResults{
				Path:             "/projects/123456/events/status/something/else",
				VisitedServerIDs: []string{"upstream"},
				UpstreamRequestHeaders: []map[string]string{{
					echo.HeaderAuthorization: "Bearer gitlabAccessTokenValue",
				}},
			},
			Tokens: []models.AuthToken{
				newTestToken(
					models.AccessTokenType,
					tokenID("accessTokenID"),
					tokenPlainValue("accessTokenValue"),
					tokenProviderID("renku"),
				),
				newTestToken(
					models.AccessTokenType,
					tokenID("gitlabAccessTokenID"),
					tokenPlainValue("gitlabAccessTokenValue"),
					tokenProviderID("gitlab"),
				),
			},
			Sessions: []models.Session{
				newTestSesssion(sessionID("sessionID"), withTokenIDs("gitlabAccessTokenID", "accessTokenID")),
			},
			RequestCookie: &http.Cookie{Name: models.SessionCookieName, Value: "sessionID"},
		},
		{
			Path:        "/api/projects/123456/graph/status",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected: TestResults{
				Path:                   "/projects/123456/events/status",
				VisitedServerIDs:       []string{"upstream"},
				UpstreamRequestHeaders: []map[string]string{{echo.HeaderAuthorization: ""}},
			},
		},
		{
			Path: "/api/projects/123456/graph/webhooks/something/else",
			Expected: TestResults{
				Path:                   "/projects/123456/webhooks/something/else",
				VisitedServerIDs:       []string{"upstream"},
				UpstreamRequestHeaders: []map[string]string{{echo.HeaderAuthorization: ""}},
			},
		},
		{
			Path:        "/api/projects/123456/graph/webhooks",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected: TestResults{
				Path:                   "/projects/123456/webhooks",
				VisitedServerIDs:       []string{"upstream"},
				UpstreamRequestHeaders: []map[string]string{{echo.HeaderAuthorization: ""}},
			},
		},
		{
			Path: "/api/datasets/test",
			Expected: TestResults{
				Path:                   "/knowledge-graph/datasets/test",
				VisitedServerIDs:       []string{"upstream"},
				UpstreamRequestHeaders: []map[string]string{{echo.HeaderAuthorization: ""}},
			},
		},
		{
			Path:        "/api/datasets",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected: TestResults{
				Path:                   "/knowledge-graph/datasets",
				VisitedServerIDs:       []string{"upstream"},
				UpstreamRequestHeaders: []map[string]string{{echo.HeaderAuthorization: ""}},
			},
		},
		{
			Path: "/api/kg/test",
			Expected: TestResults{
				Path:             "/knowledge-graph/test",
				VisitedServerIDs: []string{"upstream"},
				UpstreamRequestHeaders: []map[string]string{{
					echo.HeaderAuthorization: "Bearer gitlabAccessTokenValue",
				}},
			},
			Tokens: []models.AuthToken{
				newTestToken(
					models.AccessTokenType,
					tokenID("gitlabAccessTokenID"),
					tokenPlainValue("gitlabAccessTokenValue"),
					tokenProviderID("gitlab"),
				),
			},
			Sessions: []models.Session{
				newTestSesssion(sessionID("sessionID"), withTokenIDs("gitlabAccessTokenID")),
			},
			RequestCookie: &http.Cookie{Name: models.SessionCookieName, Value: "sessionID"},
		},
		{
			Path:        "/api/kg",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected: TestResults{
				Path:                   "/knowledge-graph",
				VisitedServerIDs:       []string{"upstream"},
				UpstreamRequestHeaders: []map[string]string{{echo.HeaderAuthorization: ""}},
			},
		},
		{
			Path: "/api/renku/test",
			Expected: TestResults{
				Path:             "/renku/test",
				VisitedServerIDs: []string{"upstream"},
				UpstreamRequestHeaders: []map[string]string{{
					echo.HeaderAuthorization: "",
					"Renku-user-id":          "",
					"Renku-user-fullname":    "",
					"renku-user-email":       "",
				}},
			},
		},
		{
			Path:        "/api/renku",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected: TestResults{
				Path:             "/renku",
				VisitedServerIDs: []string{"upstream"},
				UpstreamRequestHeaders: []map[string]string{{
					echo.HeaderAuthorization: "Bearer gitlabAccessTokenValue",
					"Renku-User":             "renkuIDTokenValue",
				}},
			},
			Tokens: []models.AuthToken{
				newTestToken(
					models.IDTokenType,
					tokenID("renkuIDTokenID"),
					tokenPlainValue("renkuIDTokenValue"),
					tokenProviderID("renku"),
				),
				newTestToken(
					models.AccessTokenType,
					tokenID("gitlabAccessTokenID"),
					tokenPlainValue("gitlabAccessTokenValue"),
					tokenProviderID("gitlab"),
				),
			},
			Sessions: []models.Session{
				newTestSesssion(sessionID("sessionID"), withTokenIDs("renkuIDTokenID", "gitlabAccessTokenID")),
			},
			RequestCookie: &http.Cookie{Name: models.SessionCookieName, Value: "sessionID"},
		},
		{
			Path:        "/api/renku/10",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected:    TestResults{Path: "/renku", VisitedServerIDs: []string{"upstream"}},
		},
		{
			Path:        "/api/renku/8",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected:    TestResults{Path: "/api/renku/8", Non200ResponseStatusCode: 404},
		},
		{
			Path:        "/api/renku/7/10/something/else",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected:    TestResults{Path: "/api/renku/7/10/something/else", Non200ResponseStatusCode: 404},
		},
		{
			Path:        "/api/renku/10/1.1/test",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected:    TestResults{Path: "/renku/1.1/test", VisitedServerIDs: []string{"upstream"}},
		},
		{
			Path:        "/api/renku/1.1/test",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected:    TestResults{Path: "/renku/1.1/test", VisitedServerIDs: []string{"upstream"}},
		},
		{
			Path:        "/api/renku/9",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected:    TestResults{Path: "/renku", VisitedServerIDs: []string{"upstream2"}},
		},
		{
			Path:        "/api/renku/9/endpoint.action",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected:    TestResults{Path: "/renku/endpoint.action", VisitedServerIDs: []string{"upstream2"}},
		},
		{
			Path:           "/gitlab/test/something",
			ExternalGitlab: true,
			Expected:       TestResults{Path: "/test/something", VisitedServerIDs: []string{"gitlab"}, UpstreamRequestHeaders: []map[string]string{{echo.HeaderAuthorization: ""}}},
		},
		{
			Path:           "/gitlab",
			ExternalGitlab: true,
			Expected:       TestResults{Path: "/", VisitedServerIDs: []string{"gitlab"}, UpstreamRequestHeaders: []map[string]string{{echo.HeaderAuthorization: ""}}},
		},
		{
			Path:           "/api/user/test/something",
			ExternalGitlab: true,
			Expected: TestResults{
				Path:             "/api/v4/user/test/something",
				VisitedServerIDs: []string{"gitlab"},
				UpstreamRequestHeaders: []map[string]string{{
					echo.HeaderAuthorization: "Bearer gitlabAccessTokenValue",
				}},
			},
			Tokens: []models.AuthToken{
				newTestToken(
					models.AccessTokenType,
					tokenID("gitlabAccessTokenID"),
					tokenPlainValue("gitlabAccessTokenValue"),
					tokenProviderID("gitlab"),
				),
			},
			Sessions: []models.Session{
				newTestSesssion(sessionID("sessionID"), withTokenIDs("gitlabAccessTokenID")),
			},
			RequestCookie: &http.Cookie{Name: models.SessionCookieName, Value: "sessionID"},
		},
		{
			Path:           "/api/user/test/something",
			ExternalGitlab: false,
			Expected: TestResults{
				Path:             "/gitlab/api/v4/user/test/something",
				VisitedServerIDs: []string{"upstream"},
				UpstreamRequestHeaders: []map[string]string{{
					echo.HeaderAuthorization: "Bearer gitlabAccessTokenValue",
				}},
			},
			Tokens: []models.AuthToken{
				newTestToken(
					models.AccessTokenType,
					tokenID("gitlabAccessTokenID"),
					tokenPlainValue("gitlabAccessTokenValue"),
					tokenProviderID("gitlab"),
				),
			},
			Sessions: []models.Session{
				newTestSesssion(sessionID("sessionID"), withTokenIDs("gitlabAccessTokenID")),
			},
			RequestCookie: &http.Cookie{Name: models.SessionCookieName, Value: "sessionID"},
		},
		{
			Path:           "/api",
			ExternalGitlab: false,
			Expected:       TestResults{Path: "/gitlab/api/v4", VisitedServerIDs: []string{"upstream"}, UpstreamRequestHeaders: []map[string]string{{echo.HeaderAuthorization: ""}}},
		},
		{
			Path:           "/api",
			ExternalGitlab: true,
			Expected:       TestResults{Path: "/api/v4", VisitedServerIDs: []string{"gitlab"}, UpstreamRequestHeaders: []map[string]string{{echo.HeaderAuthorization: ""}}},
		},
		{
			Path:           "/api/direct/test",
			ExternalGitlab: true,
			Expected:       TestResults{Path: "/test", VisitedServerIDs: []string{"gitlab"}, UpstreamRequestHeaders: []map[string]string{{echo.HeaderAuthorization: ""}}},
		},
		{
			Path:           "/api/direct",
			ExternalGitlab: true,
			Expected:       TestResults{Path: "/", VisitedServerIDs: []string{"gitlab"}, UpstreamRequestHeaders: []map[string]string{{echo.HeaderAuthorization: ""}}},
		},
		{
			Path:           "/api/direct/test",
			ExternalGitlab: false,
			Expected:       TestResults{Path: "/gitlab/test", VisitedServerIDs: []string{"upstream"}, UpstreamRequestHeaders: []map[string]string{{echo.HeaderAuthorization: ""}}},
		},
		{
			Path:           "/api/direct",
			ExternalGitlab: false,
			Expected:       TestResults{Path: "/gitlab", VisitedServerIDs: []string{"upstream"}, UpstreamRequestHeaders: []map[string]string{{echo.HeaderAuthorization: ""}}},
		},
		{
			Path:           "/api/graphql/test",
			ExternalGitlab: true,
			Expected: TestResults{
				Path:             "/api/graphql/test",
				VisitedServerIDs: []string{"gitlab"},
				UpstreamRequestHeaders: []map[string]string{{
					echo.HeaderAuthorization: "Bearer gitlabAccessTokenValue",
				}},
			},
			Tokens: []models.AuthToken{
				newTestToken(
					models.AccessTokenType,
					tokenID("gitlabAccessTokenID"),
					tokenPlainValue("gitlabAccessTokenValue"),
					tokenProviderID("gitlab"),
				),
			},
			Sessions: []models.Session{
				newTestSesssion(sessionID("sessionID"), withTokenIDs("gitlabAccessTokenID")),
			},
			RequestCookie: &http.Cookie{Name: models.SessionCookieName, Value: "sessionID"},
		},
		{
			Path:           "/api/graphql",
			ExternalGitlab: true,
			Expected:       TestResults{Path: "/api/graphql", VisitedServerIDs: []string{"gitlab"}, UpstreamRequestHeaders: []map[string]string{{echo.HeaderAuthorization: ""}}},
		},
		{
			Path:           "/api/graphql/test",
			ExternalGitlab: false,
			Expected: TestResults{
				Path:             "/gitlab/api/graphql/test",
				VisitedServerIDs: []string{"upstream"},
				UpstreamRequestHeaders: []map[string]string{{
					echo.HeaderAuthorization: "Bearer gitlabAccessTokenValue",
				}},
			},
			Tokens: []models.AuthToken{
				newTestToken(
					models.AccessTokenType,
					tokenID("gitlabAccessTokenID"),
					tokenPlainValue("gitlabAccessTokenValue"),
					tokenProviderID("gitlab"),
				),
			},
			Sessions: []models.Session{
				newTestSesssion(sessionID("sessionID"), withTokenIDs("gitlabAccessTokenID")),
			},
			RequestCookie: &http.Cookie{Name: models.SessionCookieName, Value: "sessionID"},
		},
		{
			Path:           "/api/graphql",
			ExternalGitlab: false,
			Expected:       TestResults{Path: "/gitlab/api/graphql", VisitedServerIDs: []string{"upstream"}, UpstreamRequestHeaders: []map[string]string{{echo.HeaderAuthorization: ""}}},
		},
		{
			Path:           "/repos/test",
			ExternalGitlab: true,
			Expected: TestResults{
				Path:             "/test",
				VisitedServerIDs: []string{"gitlab"},
				UpstreamRequestHeaders: []map[string]string{{
					echo.HeaderAuthorization: "Basic b2F1dGgyOmdpdGxhYkFjY2Vzc1Rva2VuVmFsdWU=", // the content of the header is base64 encoding of oauth2:gitlabAccessTokenValue
				}},
			},
			Tokens: []models.AuthToken{
				newTestToken(
					models.AccessTokenType,
					tokenID("gitlabAccessTokenID"),
					tokenPlainValue("gitlabAccessTokenValue"),
					tokenProviderID("gitlab"),
				),
			},
			Sessions: []models.Session{
				newTestSesssion(sessionID("sessionID"), withTokenIDs("gitlabAccessTokenID")),
			},
			RequestCookie: &http.Cookie{Name: models.SessionCookieName, Value: "sessionID"},
		},
		{
			Path:           "/repos",
			ExternalGitlab: true,
			Expected:       TestResults{Path: "/", VisitedServerIDs: []string{"gitlab"}, UpstreamRequestHeaders: []map[string]string{{echo.HeaderAuthorization: ""}}},
		},
		{
			Path:           "/repos/test",
			ExternalGitlab: false,
			Expected: TestResults{
				Path:             "/gitlab/test",
				VisitedServerIDs: []string{"upstream"},
				UpstreamRequestHeaders: []map[string]string{{
					echo.HeaderAuthorization: "Basic b2F1dGgyOmdpdGxhYkFjY2Vzc1Rva2VuVmFsdWU=", // the content of the header is base64 encoding of oauth2:gitlabAccessTokenValue
				}},
			},
			Tokens: []models.AuthToken{
				newTestToken(
					models.AccessTokenType,
					tokenID("gitlabAccessTokenID"),
					tokenPlainValue("gitlabAccessTokenValue"),
					tokenProviderID("gitlab"),
				),
			},
			Sessions: []models.Session{
				newTestSesssion(sessionID("sessionID"), withTokenIDs("gitlabAccessTokenID")),
			},
			RequestCookie: &http.Cookie{Name: models.SessionCookieName, Value: "sessionID"},
		},
		{
			Path:           "/repos",
			ExternalGitlab: false,
			Expected:       TestResults{Path: "/gitlab", VisitedServerIDs: []string{"upstream"}, UpstreamRequestHeaders: []map[string]string{{echo.HeaderAuthorization: ""}}},
		},
		{
			Path:           "/api/projects/some.username%2Ftest-project",
			QueryParams:    map[string]string{"statistics": "false", "doNotTrack": "true"},
			ExternalGitlab: true,
			Expected:       TestResults{Path: "/api/v4/projects/some.username%2Ftest-project", VisitedServerIDs: []string{"gitlab"}},
		},
		{
			Path:           "/api/projects/some.username%2Ftest-project",
			QueryParams:    map[string]string{"statistics": "false", "doNotTrack": "true"},
			ExternalGitlab: false,
			Expected:       TestResults{Path: "/gitlab/api/v4/projects/some.username%2Ftest-project", VisitedServerIDs: []string{"upstream"}},
		},
		{
			Path: "/api/kg/webhooks/projects/123456/events/status/something/else",
			Expected: TestResults{
				Path:             "/projects/123456/events/status/something/else",
				VisitedServerIDs: []string{"upstream"},
				UpstreamRequestHeaders: []map[string]string{{
					echo.HeaderAuthorization: "Bearer gitlabAccessTokenValue",
				}},
			},
			Tokens: []models.AuthToken{
				newTestToken(
					models.AccessTokenType,
					tokenID("gitlabAccessTokenID"),
					tokenPlainValue("gitlabAccessTokenValue"),
					tokenProviderID("gitlab"),
				),
			},
			Sessions: []models.Session{
				newTestSesssion(sessionID("sessionID"), withTokenIDs("gitlabAccessTokenID")),
			},
			RequestCookie: &http.Cookie{Name: models.SessionCookieName, Value: "sessionID"},
		},
		{
			Path:        "/api/kg/webhooks/projects/123456/events/status",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected:    TestResults{Path: "/projects/123456/events/status", VisitedServerIDs: []string{"upstream"}, UpstreamRequestHeaders: []map[string]string{{echo.HeaderAuthorization: ""}}},
		},
		{
			Path:     "/api/kg/webhooks/projects/123456/webhooks/something/else",
			Expected: TestResults{Path: "/projects/123456/webhooks/something/else", VisitedServerIDs: []string{"upstream"}, UpstreamRequestHeaders: []map[string]string{{echo.HeaderAuthorization: ""}}},
		},
		{
			Path:        "/api/kg/webhooks/projects/123456/webhooks",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected:    TestResults{Path: "/projects/123456/webhooks", VisitedServerIDs: []string{"upstream"}, UpstreamRequestHeaders: []map[string]string{{echo.HeaderAuthorization: ""}}},
		},
		{
			Path:     "/api/kc/auth/realms/Renku/protocol/openid-connect/userinfo",
			Expected: TestResults{Path: "/auth/realms/Renku/protocol/openid-connect/userinfo", VisitedServerIDs: []string{"upstream"}},
		},
	}
	for _, testCase := range testCases {
		// Test names show up poorly in vscode if the name contains "/"
		t.Run(strings.ReplaceAll(testCase.Path, "/", "|"), ParametrizedRouteTest(testCase))
	}
}
