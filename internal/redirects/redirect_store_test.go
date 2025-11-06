package redirects

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	netUrl "net/url"
	"path"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/labstack/echo/v4"
)

func newMockRenkuDataService(t *testing.T, calls *int32) *httptest.Server {
	// Test server that returns a JSON redirect for any requested escaped URL
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(calls, 1)
		// Expect path like /api/data/platform/redirects/<escaped>
		prefix := "/api/data/platform/redirects/"
		if !strings.HasPrefix(r.URL.Path, prefix) {
			http.NotFound(w, r)
			return
		}
		escaped := strings.TrimPrefix(r.URL.Path, prefix)
		src, _ := netUrl.QueryUnescape(escaped)
		resp := PlatformRedirectConfig{
			SourceUrl: src,
			TargetUrl: "https://renku.example.org/some/path",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	return ts
}

func TestNewRedirectStoreConfigDefaultsAndValidation(t *testing.T) {
	// Missing RenkuBaseURL should return an error
	_, err := NewRedirectStore(WithConfig(config.RedirectsStoreConfig{}))
	if err == nil {
		t.Fatal("expected error when RenkuBaseURL is not provided")
	}

	// With RenkuBaseURL provided, should succeed and set default redirectedHost
	u, _ := netUrl.Parse("https://renku.example.org")
	rs, err := NewRedirectStore(WithConfig(config.RedirectsStoreConfig{RenkuBaseURL: u}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rs.PathPrefix != "/api/gitlab-redirect/" {
		t.Fatalf("unexpected default PathPrefix: got %q", rs.PathPrefix)
	}
	if rs.redirectedHost == "" {
		t.Fatalf("expected redirectedHost to be set to a default, got empty")
	}
}

func TestGetRedirectEntry(t *testing.T) {
	var calls int32
	ts := newMockRenkuDataService(t, &calls)
	defer ts.Close()

	// Use the test server's client for DefaultClient during this test and restore afterwards
	origDefaultClient := http.DefaultClient
	http.DefaultClient = ts.Client()
	t.Cleanup(func() { http.DefaultClient = origDefaultClient })

	// Configure RedirectStore to point to the test server host
	u, _ := netUrl.Parse(ts.URL)
	cfg := config.RedirectsStoreConfig{RenkuBaseURL: u}
	rs, err := NewRedirectStore(WithConfig(cfg), WithEntryTtl(1*time.Minute))
	if err != nil {
		t.Fatalf("failed to create RedirectStore: %v", err)
	}

	full := path.Join(rs.PathPrefix, "user/repo")

	// First call should hit the server
	e1, err := rs.GetRedirectEntry(netUrl.URL{Path: full})
	if err != nil {
		t.Fatalf("GetRedirectEntry returned error: %v", err)
	}
	if e1 == nil {
		t.Fatalf("expected an entry with TargetUrl, got %+v", e1)
	}
	if e1.TargetUrl != "https://renku.example.org/some/path" {
		t.Fatalf("unexpected target url: %v", e1.TargetUrl)
	}

	// Second call (within TTL) should be served from cache -> server not called again
	e2, err := rs.GetRedirectEntry(netUrl.URL{Path: full})
	if err != nil {
		t.Fatalf("GetRedirectEntry (second) returned error: %v", err)
	}
	if e2 == nil {
		t.Fatalf("expected a cached entry with TargetUrl, got %+v", e2)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected server to be called once, was called %d times", calls)
	}
}

func TestRedirectStoreMiddleware(t *testing.T) {
	var calls int32
	ts := newMockRenkuDataService(t, &calls)
	defer ts.Close()

	origDefaultClient := http.DefaultClient
	http.DefaultClient = ts.Client()
	t.Cleanup(func() { http.DefaultClient = origDefaultClient })

	u, _ := netUrl.Parse(ts.URL)
	cfg := config.RedirectsStoreConfig{RenkuBaseURL: u}
	rs, err := NewRedirectStore(WithConfig(cfg), WithEntryTtl(1*time.Minute))
	if err != nil {
		t.Fatalf("failed to create RedirectStore: %v", err)
	}

	e := echo.New()

	// Build a request that matches the PathPrefix and will be transformed by the middleware
	req := httptest.NewRequest("GET", path.Join(rs.PathPrefix, "group/project"), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Create middleware and execute it; next should not be called because a redirect exists
	mw := rs.Middleware()
	nextCalled := false
	next := func(c echo.Context) error {
		nextCalled = true
		return c.String(200, "next")
	}

	if err := mw(next)(c); err != nil {
		t.Fatalf("middleware returned error: %v", err)
	}

	if nextCalled {
		t.Fatalf("expected middleware to handle redirect and not call next")
	}
	if rec.Result().StatusCode != http.StatusMovedPermanently {
		t.Fatalf("expected redirect status %d, got %d", http.StatusMovedPermanently, rec.Result().StatusCode)
	}
	loc := rec.Header().Get("Location")
	if loc != "https://renku.example.org/some/path" {
		t.Fatalf("unexpected redirect location: %q", loc)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected server to be called once, was called %d times", calls)
	}
}
