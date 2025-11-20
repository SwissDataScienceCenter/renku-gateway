package redirects

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
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
	Config     config.RedirectsStoreConfig
	PathPrefix string

	entryTtl         time.Duration
	redirectMap      map[string]RedirectStoreRedirectEntry
	redirectedHost   string
	redirectMapMutex sync.Mutex
}

type RedirectStoreOption func(*RedirectStore) error

func WithConfig(cfg config.RedirectsStoreConfig) RedirectStoreOption {
	return func(rs *RedirectStore) error {
		rs.Config = cfg
		return nil
	}
}

func queryRenkuApi(ctx context.Context, host url.URL, endpoint string) ([]byte, error) {

	rel, err := url.Parse("/api/data")
	if err != nil {
		return nil, fmt.Errorf("error parsing endpoint: %w", err)
	}
	rel = rel.JoinPath(endpoint)
	fullUrl := host.ResolveReference(rel).String()
	req, err := http.NewRequestWithContext(ctx, "GET", fullUrl, nil)
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

func retrieveRedirectTargetForSource(ctx context.Context, host url.URL, source string) (*PlatformRedirectConfig, error) {
	// Query the Renku API to get the redirect for the given source URL
	body, err := queryRenkuApi(ctx, host, fmt.Sprintf("/platform/redirects/%s", source))
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

func (rs *RedirectStore) urlToKey(redirectUrl url.URL) (string, error) {

	path := redirectUrl.Path
	if path == "" || !strings.HasPrefix(path, rs.PathPrefix) {
		return "", fmt.Errorf("the path should start with the prefix %s", rs.PathPrefix)
	}

	urlToCheck := strings.TrimPrefix(path, rs.PathPrefix)
	// TODO: Check for a `/-/` in the path and remove it and anything that follows (links to sub-pages of a project)
	urlToCheck = fmt.Sprintf("https://%s/%s", rs.redirectedHost, urlToCheck)
	// URL-encode the full URL so it can be safely used in the API path
	urlToCheck = url.QueryEscape(urlToCheck)
	// check for redirects for this URL
	return urlToCheck, nil
}

func (rs *RedirectStore) GetRedirectEntry(ctx context.Context, url url.URL) (*RedirectStoreRedirectEntry, error) {
	key, err := rs.urlToKey(url)
	if err != nil {
		return nil, fmt.Errorf("error converting url to key: %w", err)
	}

	entry, ok := rs.redirectMap[key]
	if ok && entry.UpdatedAt.Add(rs.entryTtl).After(time.Now()) {
		return &entry, nil
	}

	rs.redirectMapMutex.Lock()
	defer rs.redirectMapMutex.Unlock()
	// Re-check after acquiring the lock, since it might have been updated meanwhile
	entry, ok = rs.redirectMap[key]
	if !ok || entry.UpdatedAt.Add(rs.entryTtl).Before(time.Now()) {
		updatedEntry, err := retrieveRedirectTargetForSource(ctx, *rs.Config.Gitlab.RenkuBaseURL, key)
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
		rs.redirectMap[key] = entry
	}
	return &entry, nil
}

func (rs *RedirectStore) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			redirectUrl := c.Request().URL
			if redirectUrl == nil {
				return next(c)
			}
			ctx := c.Request().Context()
			// check for redirects for this URL
			entry, err := rs.GetRedirectEntry(ctx, *redirectUrl)

			if err != nil {
				slog.Debug(
					"REDIRECT_STORE MIDDLEWARE",
					"message",
					"could not lookup redirect entry, returning 404",
					"url",
					redirectUrl.String(),
					"error",
					err.Error(),
				)
				return c.NoContent(http.StatusNotFound)
			}
			if entry == nil {
				slog.Debug(
					"REDIRECT_STORE MIDDLEWARE",
					"message", "nil redirect found for url (this should not happen), returning 404",
					"from", redirectUrl.String(),
				)
				return c.NoContent(http.StatusNotFound)
			}
			if entry == &noRedirectFound {
				slog.Debug(
					"REDIRECT_STORE MIDDLEWARE",
					"message", "no redirect found for url, returning 404",
					"from", redirectUrl.String(),
				)
				return c.NoContent(http.StatusNotFound)
			}
			slog.Debug(
				"REDIRECT_STORE MIDDLEWARE",
				"message", "redirecting request",
				"from", redirectUrl.String(),
				"to", entry.TargetUrl,
			)
			return c.Redirect(http.StatusMovedPermanently, entry.TargetUrl)
		}
	}
}

func NewRedirectStore(options ...RedirectStoreOption) (*RedirectStore, error) {
	rs := RedirectStore{redirectMap: make(map[string]RedirectStoreRedirectEntry), PathPrefix: "/api/gitlab-redirect/", redirectMapMutex: sync.Mutex{}}
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

	rs.entryTtl = time.Duration(rs.Config.Gitlab.EntryTtlSeconds) * time.Second

	return &rs, nil
}
