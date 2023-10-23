package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

const serverIDHeader string = "Server-ID"

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

func setupTestAuthServer(ID string, responseHeaders map[string]string, responseStatus int, requestTracker chan<- *http.Request) (*httptest.Server, *url.URL) {
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

func setupTestRevproxy(ctx context.Context, upstreamServerURL *url.URL, upstreamServerURL2 *url.URL, authURL *url.URL, externalGitlabURL *url.URL) (*echo.Echo, *url.URL) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	config := revProxyConfig{
		RenkuBaseURL:      upstreamServerURL,
		ExternalGitlabURL: externalGitlabURL,
		Port:              port,
		RenkuServices: renkuServicesConfig{
			Notebooks:        upstreamServerURL,
			CoreServiceNames: []string{upstreamServerURL.String(), upstreamServerURL.String(), upstreamServerURL2.String()},
			CoreServicePaths: []string{"/api/renku", "/api/renku/10", "/api/renku/9"},
			KG:               upstreamServerURL,
			Webhook:          upstreamServerURL,
			Auth:             authURL,
			Crc: 	          upstreamServerURL,
		},
		Debug: true,
	}
	proxy := setupServer(ctx, config)
	go func() {
		proxy.Listener = listener
		err := proxy.Start(fmt.Sprintf(":%d", config.Port))
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
		defer listener.Close()
	}()
	url, err := url.Parse(fmt.Sprintf("http://localhost:%d", config.Port))
	if err != nil {
		log.Fatal(err)
	}
	return proxy, url
}

type TestResults struct {
	Path                     string
	VisitedServerIDs         []string
	ResponseHeaders          map[string]string
	Non200ResponseStatusCode int
	IgnoreErrors             bool
}

type TestCase struct {
	Path                         string
	QueryParams                  map[string]string
	Non200AuthResponseStatusCode int
	ExternalGitlab               bool
	Expected                     TestResults
}

func ParametrizedRouteTest(scenario TestCase) func(*testing.T) {
	return func(t *testing.T) {
		// Setup and start
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		requestTracker := make(testRequestTracker, 20)
		upstream, upstreamURL := setupTestUpstream("upstream", requestTracker)
		upstream2, upstreamURL2 := setupTestUpstream("upstream2", requestTracker)
		var authResponseStatusCode int
		if scenario.Non200AuthResponseStatusCode == 0 {
			authResponseStatusCode = http.StatusOK
		} else {
			authResponseStatusCode = scenario.Non200AuthResponseStatusCode
		}
		auth, authURL := setupTestAuthServer("auth", map[string]string{"Authorization": "secret-token"}, authResponseStatusCode, requestTracker)
		var (
			gitlab    *httptest.Server
			gitlabURL *url.URL
		)
		if scenario.ExternalGitlab {
			gitlab, gitlabURL = setupTestUpstream("gitlab", requestTracker)
			defer gitlab.Close()
		}
		proxy, proxyURL := setupTestRevproxy(shutdownCtx, upstreamURL, upstreamURL2, authURL, gitlabURL)
		defer upstream.Close()
		defer upstream2.Close()
		defer proxy.Close()
		defer auth.Close()

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
		res, err := httpClient.Get(reqURL.String())
		assert.NoError(t, err)
		reqs := requestTracker.getAllRequests()

		// Assert the request was routed as expected
		if !scenario.Expected.IgnoreErrors {
			assert.NoError(t, err)
		}
		if scenario.Expected.Non200ResponseStatusCode != 0 {
			assert.Equal(t, scenario.Expected.Non200ResponseStatusCode, res.StatusCode)
		} else {
			assert.Equal(t, http.StatusOK, res.StatusCode)
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
		if len(scenario.QueryParams) > 0 && len(reqs) > 0  {
			assert.Equal(t, reqURLQuery.Encode(), reqs[len(reqs)-1].URL.RawQuery)
		}
	}
}

func TestInternalSvcRoutes(t *testing.T) {
	testCases := []TestCase{
		{
			Path:     "/api/auth/test",
			Expected: TestResults{Path: "/api/auth/test", VisitedServerIDs: []string{"auth"}},
		},
		{
			Path:     "/api/auth",
			Expected: TestResults{Path: "/api/auth", VisitedServerIDs: []string{"auth"}},
		},
		{
			Path:                         "/api/notebooks/test/rejectedAuth",
			Non200AuthResponseStatusCode: 401,
			Expected:                     TestResults{VisitedServerIDs: []string{"auth"}, Non200ResponseStatusCode: 401},
		},
		{
			Path:     "/api/notebooks/test/acceptedAuth",
			Expected: TestResults{Path: "/notebooks/test/acceptedAuth", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:     "/api/notebooks",
			Expected: TestResults{Path: "/notebooks", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:     "/api/projects/123456/graph/status/something/else",
			Expected: TestResults{Path: "/projects/123456/events/status/something/else", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:        "/api/projects/123456/graph/status",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected:    TestResults{Path: "/projects/123456/events/status", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:     "/api/projects/123456/graph/webhooks/something/else",
			Expected: TestResults{Path: "/projects/123456/webhooks/something/else", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:        "/api/projects/123456/graph/webhooks",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected:    TestResults{Path: "/projects/123456/webhooks", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:     "/api/datasets/test",
			Expected: TestResults{Path: "/knowledge-graph/datasets/test", VisitedServerIDs: []string{"upstream"}},
		},
		{
			Path:        "/api/datasets",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected:    TestResults{Path: "/knowledge-graph/datasets", VisitedServerIDs: []string{"upstream"}},
		},
		{
			Path:     "/api/kg/test",
			Expected: TestResults{Path: "/knowledge-graph/test", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:        "/api/kg",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected:    TestResults{Path: "/knowledge-graph", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:     "/api/renku/test",
			Expected: TestResults{Path: "/renku/test", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:        "/api/renku",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected:    TestResults{Path: "/renku", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:        "/api/renku/10",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected:    TestResults{Path: "/renku", VisitedServerIDs: []string{"auth", "upstream"}},
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
			Expected:    TestResults{Path: "/renku/1.1/test", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:        "/api/renku/1.1/test",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected:    TestResults{Path: "/renku/1.1/test", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:        "/api/renku/9",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected:    TestResults{Path: "/renku", VisitedServerIDs: []string{"auth", "upstream2"}},
		},
		{
			Path:        "/api/renku/9/endpoint.action",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected:    TestResults{Path: "/renku/endpoint.action", VisitedServerIDs: []string{"auth", "upstream2"}},
		},
		{
			Path:           "/gitlab/test/something",
			ExternalGitlab: true,
			Expected:       TestResults{Path: "/test/something", VisitedServerIDs: []string{"gitlab"}},
		},
		{
			Path:           "/gitlab",
			ExternalGitlab: true,
			Expected:       TestResults{Path: "/", VisitedServerIDs: []string{"gitlab"}},
		},
		{
			Path:           "/api/user/test/something",
			ExternalGitlab: true,
			Expected:       TestResults{Path: "/api/v4/user/test/something", VisitedServerIDs: []string{"auth", "gitlab"}},
		},
		{
			Path:           "/api/user/test/something",
			ExternalGitlab: false,
			Expected:       TestResults{Path: "/gitlab/api/v4/user/test/something", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:           "/api",
			ExternalGitlab: false,
			Expected:       TestResults{Path: "/gitlab/api/v4", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:           "/api",
			ExternalGitlab: true,
			Expected:       TestResults{Path: "/api/v4", VisitedServerIDs: []string{"auth", "gitlab"}},
		},
		{
			Path:           "/api/direct/test",
			ExternalGitlab: true,
			Expected:       TestResults{Path: "/test", VisitedServerIDs: []string{"gitlab"}},
		},
		{
			Path:           "/api/direct",
			ExternalGitlab: true,
			Expected:       TestResults{Path: "/", VisitedServerIDs: []string{"gitlab"}},
		},
		{
			Path:           "/api/direct/test",
			ExternalGitlab: false,
			Expected:       TestResults{Path: "/gitlab/test", VisitedServerIDs: []string{"upstream"}},
		},
		{
			Path:           "/api/direct",
			ExternalGitlab: false,
			Expected:       TestResults{Path: "/gitlab", VisitedServerIDs: []string{"upstream"}},
		},
		{
			Path:           "/api/graphql/test",
			ExternalGitlab: true,
			Expected:       TestResults{Path: "/api/graphql/test", VisitedServerIDs: []string{"auth", "gitlab"}},
		},
		{
			Path:           "/api/graphql",
			ExternalGitlab: true,
			Expected:       TestResults{Path: "/api/graphql", VisitedServerIDs: []string{"auth", "gitlab"}},
		},
		{
			Path:           "/api/graphql/test",
			ExternalGitlab: false,
			Expected:       TestResults{Path: "/gitlab/api/graphql/test", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:           "/api/graphql",
			ExternalGitlab: false,
			Expected:       TestResults{Path: "/gitlab/api/graphql", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:           "/repos/test",
			ExternalGitlab: true,
			Expected:       TestResults{Path: "/test", VisitedServerIDs: []string{"auth", "gitlab"}},
		},
		{
			Path:           "/repos",
			ExternalGitlab: true,
			Expected:       TestResults{Path: "/", VisitedServerIDs: []string{"auth", "gitlab"}},
		},
		{
			Path:           "/repos/test",
			ExternalGitlab: false,
			Expected:       TestResults{Path: "/gitlab/test", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:           "/repos",
			ExternalGitlab: false,
			Expected:       TestResults{Path: "/gitlab", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:           "/api/projects/some.username%2Ftest-project",
			QueryParams:    map[string]string{"statistics": "false", "doNotTrack": "true"},
			ExternalGitlab: true,
			Expected:       TestResults{Path: "/api/v4/projects/some.username%2Ftest-project", VisitedServerIDs: []string{"auth", "gitlab"}},
		},
		{
			Path:           "/api/projects/some.username%2Ftest-project",
			QueryParams:    map[string]string{"statistics": "false", "doNotTrack": "true"},
			ExternalGitlab: false,
			Expected:       TestResults{Path: "/gitlab/api/v4/projects/some.username%2Ftest-project", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:     "/api/kg/webhooks/projects/123456/events/status/something/else",
			Expected: TestResults{Path: "/projects/123456/events/status/something/else", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:        "/api/kg/webhooks/projects/123456/events/status",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected:    TestResults{Path: "/projects/123456/events/status", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:     "/api/kg/webhooks/projects/123456/webhooks/something/else",
			Expected: TestResults{Path: "/projects/123456/webhooks/something/else", VisitedServerIDs: []string{"auth", "upstream"}},
		},
		{
			Path:        "/api/kg/webhooks/projects/123456/webhooks",
			QueryParams: map[string]string{"test1": "value1", "test2": "value2"},
			Expected:    TestResults{Path: "/projects/123456/webhooks", VisitedServerIDs: []string{"auth", "upstream"}},
		},
	}
	for _, testCase := range testCases {
		// Test names show up poorly in vscode if the name contains "/"
		t.Run(strings.ReplaceAll(testCase.Path, "/", "|"), ParametrizedRouteTest(testCase))
	}
}
