package redirects

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	netUrl "net/url"
	"strings"
	"sync"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/labstack/echo/v4"
)

type PlatformRedirectConfig struct {
	SourceUrl string `json:"source_url"`
	TargetUrl string `json:"target_url"`
}

type RedirectStoreRedirectEntry struct {
	SourceUrl string
	TargetUrl string
	UpdatedAt time.Time
}

var noRedirectFound = RedirectStoreRedirectEntry{}

type RedirectStore struct {
	Config      config.RedirectsStoreConfig
	EntryTtl    time.Duration
	RedirectMap map[string]RedirectStoreRedirectEntry

	PathPrefix string

	redirectedHost   string
	redirectMapMutex sync.Mutex
}

type ServerCredentials struct {
	Host netUrl.URL
}

type RedirectStoreOption func(*RedirectStore) error

func WithConfig(cfg config.RedirectsStoreConfig) RedirectStoreOption {
	return func(rs *RedirectStore) error {
		rs.Config = cfg
		return nil
	}
}

func WithEntryTtl(ttl time.Duration) RedirectStoreOption {
	return func(rs *RedirectStore) error {
		rs.EntryTtl = ttl
		return nil
	}
}

func queryRenkuApi(renkuCredentials ServerCredentials, endpoint string) ([]byte, error) {
	method := "GET"

	path := fmt.Sprintf("/api/data%s", endpoint)
	rel, err := netUrl.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("error parsing endpoint: %w", err)
	}
	fullUrl := renkuCredentials.Host.ResolveReference(rel).String()
	req, err := http.NewRequest(method, fullUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching migrated projects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	return body, nil
}

func retrieveRedirectTargetForSource(renkuCredentials ServerCredentials, source string) (*PlatformRedirectConfig, error) {
	// Query the Renku API to get the redirect for the given source URL
	body, err := queryRenkuApi(renkuCredentials, fmt.Sprintf("/platform/redirects/%s", source))
	if err != nil {
		return nil, fmt.Errorf("error querying Renku API: %w", err)
	}
	if body == nil {
		return nil, fmt.Errorf("no response body received")
	}

	var redirectConfig PlatformRedirectConfig
	if err := json.Unmarshal(body, &redirectConfig); err != nil {
		return nil, fmt.Errorf("error parsing JSON response: %w", err)
	}

	return &redirectConfig, nil
}

func (rs *RedirectStore) urlToKey(url netUrl.URL) (string, error) {

	path := url.Path
	if path == "" {
		return "", fmt.Errorf("the path should start with PathPrefix")
	}
	if !strings.HasPrefix(path, rs.PathPrefix) {
		return "", fmt.Errorf("the path should start with PathPrefix")
	}

	urlToCheck := strings.TrimPrefix(path, rs.PathPrefix)
	// TODO: Check for a `/-/` in the path and remove it and anything that follows (links to sub-pages of a project)
	urlToCheck = fmt.Sprintf("https://%s/%s", rs.redirectedHost, urlToCheck)
	// URL-encode the full URL so it can be safely used in the API path
	urlToCheck = netUrl.QueryEscape(urlToCheck)
	// check for redirects for this URL
	return urlToCheck, nil
}

func (rs *RedirectStore) GetRedirectEntry(url netUrl.URL) (*RedirectStoreRedirectEntry, error) {
	if rs == nil {
		return nil, fmt.Errorf("redirect store is not initialized")
	}

	key, err := rs.urlToKey(url)
	if err != nil {
		return nil, fmt.Errorf("error converting url to key: %w", err)
	}

	entry, ok := rs.RedirectMap[key]
	if ok && entry.UpdatedAt.Add(rs.EntryTtl).After(time.Now()) {
		return &entry, nil
	}

	rs.redirectMapMutex.Lock()
	defer rs.redirectMapMutex.Unlock()
	// Re-check after acquiring the lock, since it might have been updated meanwhile
	entry, ok = rs.RedirectMap[key]
	if !ok || entry.UpdatedAt.Add(rs.EntryTtl).Before(time.Now()) {
		updatedEntry, err := retrieveRedirectTargetForSource(ServerCredentials{
			Host: *rs.Config.Gitlab.RenkuBaseURL, // RenkuBaseURL cannot be non-nil here due to earlier validation
		}, key)
		if err != nil {
			return nil, fmt.Errorf("error retrieving redirect for url %s: %w", key, err)
		}
		if updatedEntry == nil {
			// No entry, this is fine
			return &noRedirectFound, nil
		}
		entry = RedirectStoreRedirectEntry{
			SourceUrl: updatedEntry.SourceUrl,
			TargetUrl: updatedEntry.TargetUrl,
			UpdatedAt: time.Now(),
		}
		rs.RedirectMap[key] = entry
	}
	return &entry, nil
}

func (rs *RedirectStore) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			url := c.Request().URL
			if url == nil {
				return next(c)
			}
			// check for redirects for this URL
			entry, err := rs.GetRedirectEntry(*url)

			if err != nil {
				slog.Debug(
					"REDIRECT_STORE MIDDLEWARE",
					"message",
					"could not lookup redirect entry, returning 404",
					"url",
					url.String(),
					"error",
					err.Error(),
				)
				return c.NoContent(http.StatusNotFound)
			}
			if entry == nil {
				slog.Debug(
					"REDIRECT_STORE MIDDLEWARE",
					"message", "nil redirect found for url (this should not happen), returning 404",
					"from", url.String(),
				)
				return c.NoContent(http.StatusNotFound)
			}
			if entry == &noRedirectFound {
				slog.Debug(
					"REDIRECT_STORE MIDDLEWARE",
					"message", "no redirect found for url, returning 404",
					"from", url.String(),
				)
				return c.NoContent(http.StatusNotFound)
			}
			slog.Debug(
				"REDIRECT_STORE MIDDLEWARE",
				"message", "redirecting request",
				"from", url.String(),
				"to", entry.TargetUrl,
			)
			return c.Redirect(http.StatusMovedPermanently, entry.TargetUrl)
		}
	}
}

func NewRedirectStore(options ...RedirectStoreOption) (*RedirectStore, error) {
	rs := RedirectStore{RedirectMap: make(map[string]RedirectStoreRedirectEntry), PathPrefix: "/api/gitlab-redirect/", redirectMapMutex: sync.Mutex{}}
	for _, opt := range options {
		err := opt(&rs)
		if err != nil {
			return &RedirectStore{}, err
		}
	}

	if !rs.Config.Gitlab.Enabled {
		return nil, nil
	}

	if rs.Config.Gitlab.RenkuBaseURL == nil {
		return &RedirectStore{}, fmt.Errorf("a RenkuBaseURL must be provided")
	}

	rs.redirectedHost = rs.Config.Gitlab.RedirectedHost
	if rs.redirectedHost == "" {
		rs.redirectedHost = "gitlab.renkulab.io"
	}

	return &rs, nil
}
