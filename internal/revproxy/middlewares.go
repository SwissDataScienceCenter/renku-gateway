package revproxy

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/utils"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// noCookies middleware removes all cookies from a request
func noCookies(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Request().Header.Set("cookie", "")
		return next(c)
	}
}

// regexRewrite is a small helper function to produce a path rewrite middleware
func regexRewrite(match, replace string) echo.MiddlewareFunc {
	config := middleware.RewriteConfig{
		RegexRules: map[*regexp.Regexp]string{
			regexp.MustCompile(match): replace,
		},
	}
	return middleware.RewriteWithConfig(config)
}

// stripPrefix middleware removes a prefix from a request's path
func stripPrefix(prefix string) echo.MiddlewareFunc {
	return middleware.RewriteWithConfig(middleware.RewriteConfig{
		RegexRules: map[*regexp.Regexp]string{
			regexp.MustCompile(fmt.Sprintf("^%s/(.+)", prefix)): "/$1",
			regexp.MustCompile(fmt.Sprintf("^%s$", prefix)):     "/",
		},
	})
}

// setHost middleware sets the host field and header of a request. Needed to make
// proxying to external services work. Withtout this middleware proxying to
// anything outside of the cluster fails.
func setHost(host string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Request().Host = host
			return next(c)
		}
	}
}

// ensureSession middleware makes sure a session exists by creating a new one if none is found.
func ensureSession(sessions *sessions.SessionStore) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			session, err := sessions.Get(c)
			if err != nil || session.ID == "" {
				_, err = sessions.Create(c)
			}
			if err != nil {
				return err
			}
			return next(c)
		}
	}
}

// gitlabRedirect redirects from the old-style internal gitlab url to an external Gitlab instance
func gitlabRedirect(newGitlabHost string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			oldURL := c.Request().URL
			newURL := *oldURL
			newURL.Host = newGitlabHost
			newURL.Path = strings.TrimPrefix(newURL.Path, "/gitlab")
			newURL.RawPath = strings.TrimPrefix(newURL.RawPath, "/gitlab")
			return c.Redirect(http.StatusMovedPermanently, newURL.String())
		}
	}
}

// uiServerUpstreamInternalGitlabLocation is used to set headers used by the UI server to route 1 specific request for
// Gitlab, when a Renku-bundled Gitlab is used. The UI server needs to cache or further process the results from
// this reqest, therefore it is not possible to fully skip the UI server.
func uiServerUpstreamInternalGitlabLocation(host string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			upstreamPath := *c.Request().URL
			upstreamPath.Host = ""
			upstreamPath.Scheme = ""
			upstreamPathStr := strings.TrimPrefix(upstreamPath.String(), "/ui-server/api")
			c.Request().Header.Set("Renku-Gateway-Upstream-Path", "/gitlab/api/v4"+upstreamPathStr)
			c.Request().Header.Set("Renku-Gateway-Upstream-Host", host)
			return next(c)
		}
	}
}

// uiServerUpstreamExternalGitlabLocation is used to set headers used by the UI server to route 1 specific request for
// Gitlab, when an external Gitlab is used with Renku. The UI server needs to cache or further process the results from
// this reqest, therefore it is not possible to fully skip the UI server.
func uiServerUpstreamExternalGitlabLocation(host string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			upstreamPath := *c.Request().URL
			upstreamPath.Host = ""
			upstreamPath.Scheme = ""
			upstreamPathStr := strings.TrimPrefix(upstreamPath.String(), "/ui-server/api")
			c.Request().Header.Set("Renku-Gateway-Upstream-Path", "/api/v4"+upstreamPathStr)
			c.Request().Header.Set("Renku-Gateway-Upstream-Host", host)
			return next(c)
		}
	}
}

// uiServerPathRewrite changes the incoming requests so that the UI server is used (as a second proxy) only for very
// specific endpoints (when absolutely necessary). For all other cases the gateway routes directly to the required
// Renku component and injects the proper credentials required by the specific component.
func UiServerPathRewrite() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path
			// For several endpoints below the gateway still cannot skip the UI server.
			if strings.HasPrefix(path, "/ui-server/api/allows-iframe") ||
				strings.HasPrefix(path, "/ui-server/api/projects") ||
				strings.HasPrefix(path, "/ui-server/api/renku/cache.files_upload") ||
				strings.HasPrefix(path, "/ui-server/api/last-projects") ||
				strings.HasPrefix(path, "/ui-server/api/last-searches") {
				return next(c)
			}
			// Rewrite for /ui-server/auth -> /api/auth.
			if strings.HasPrefix(path, "/ui-server/auth") {
				originalURL := c.Request().URL.String()
				c.Request().URL.Path = "/api" + strings.TrimPrefix(path, "/ui-server")
				c.Request().URL.RawPath = "/api" + strings.TrimPrefix(c.Request().URL.RawPath, "/ui-server")
				newUrl, err := url.Parse(c.Request().URL.String())
				if err != nil {
					return err
				}
				c.Request().URL = newUrl
				c.Request().RequestURI = newUrl.String()
				slog.Debug("PATH REWRITE", "message", "matched /ui-server/auth", "originalURL", originalURL, "newUrl", newUrl.String(), "requestID", utils.GetRequestID(c))
			}
			// For notebooks rewrite to go to the data service
			if strings.HasPrefix(path, "/ui-server/api/notebooks") {
				originalURL := c.Request().URL.String()
				c.Request().URL.Path = strings.TrimPrefix(path, "/ui-server/api/notebooks")
				c.Request().URL.RawPath = strings.TrimPrefix(c.Request().URL.RawPath, "/ui-server/api/notebooks")
				c.Request().URL.Path = "/api/data/notebooks" + c.Request().URL.Path
				c.Request().URL.RawPath = "/api/data/notebooks" + c.Request().URL.RawPath
				newUrl, err := url.Parse(c.Request().URL.String())
				if err != nil {
					return err
				}
				c.Request().URL = newUrl
				c.Request().RequestURI = newUrl.String()
				slog.Debug("PATH REWRITE", "message", "matched /ui-server/api/notebooks", "originalURL", originalURL, "newUrl", newUrl.String(), "requestID", utils.GetRequestID(c))
			}
			// For all other endpoints the gateway will fully bypass the UI server routing things directly to the proper
			// Renku component.
			if strings.HasPrefix(path, "/ui-server/api") {
				originalURL := c.Request().URL.String()
				c.Request().URL.Path = strings.TrimPrefix(path, "/ui-server")
				c.Request().URL.RawPath = strings.TrimPrefix(c.Request().URL.RawPath, "/ui-server")
				newUrl, err := url.Parse(c.Request().URL.String())
				if err != nil {
					return err
				}
				c.Request().URL = newUrl
				c.Request().RequestURI = newUrl.String()
				slog.Debug("PATH REWRITE", "message", "matched /ui-server/api", "originalURL", originalURL, "newUrl", newUrl.String(), "requestID", utils.GetRequestID(c))
			}
			return next(c)
		}
	}
}

// Injects the sessionID as an identifier for an anonymous user. It will only do so if there are no
// headers already injected for the keycloak tokens. Therefore this should always run in the middleware
// chain after all other token injection middelwares have run.
func notebooksAnonymousID(sessions *sessions.SessionStore) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// The request is not anonymous, continue
			if c.Request().Header.Get("Renku-Auth-Access-Token") != "" || c.Request().Header.Get("Renku-Auth-Id-Token") != "" || c.Request().Header.Get("Renku-Auth-Refresh-Token") != "" {
				return next(c)
			}
			// Use the session ID as the anonymous ID
			session, err := sessions.Get(c)
			if err != nil || session.ID == "" {
				session, err = sessions.Create(c)
			}
			if err != nil {
				return err
			}
			// NOTE: The anonymous session ID must start with a letter, otherwise when we use it to create sessions in k8s
			// things fail because a label value must start with a letter. That is why we add `anon-` here to the value.
			// Note that valid values for a label in k8s are [a-zA-Z0-9], also -_. and it must start with a letter.
			c.Request().Header.Set("Renku-Auth-Anon-Id", "anon-"+session.ID)
			return next(c)
		}
	}
}
