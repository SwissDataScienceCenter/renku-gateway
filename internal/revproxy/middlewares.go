package revproxy

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// noCookies middleware removes all cookies from a request
func noCookies(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Request().Header.Set(http.CanonicalHeaderKey("cookies"), "")
		return next(c)
	}
}

// stripCookies removes all cookies from a request, except for those provided in the keepCookies list
func stripCookies(keepCookies []string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			originalReq := c.Request().Clone(c.Request().Context())
			c.Request().Header.Set(http.CanonicalHeaderKey("cookies"), "")
			for _, cookieNameToAdd := range keepCookies {
				cookieToAdd, err := originalReq.Cookie(cookieNameToAdd)
				if err == nil {
					c.Request().AddCookie(cookieToAdd)
				}
			}
			return next(c)
		}
	}
}

// injectCredentials middleware makes a call to authenticate the request and injects the credentials
// if the authentication is successful, if not it returns the response from the autnetication service
func authenticate(authURL *url.URL, injectedHeaders ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Send request to the authorization service
			req, err := http.NewRequestWithContext(
				c.Request().Context(),
				"GET",
				authURL.String(),
				nil,
			)
			if err != nil {
				return err
			}
			req.Header = c.Request().Header.Clone()
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			// The authentication request was rejected, return the authentication service response and status code
			if res.StatusCode >= 300 || res.StatusCode < 200 {
				defer res.Body.Close()
				for name, values := range res.Header {
					c.Response().Header()[name] = values
				}
				c.Response().WriteHeader(res.StatusCode)
				_, err = io.Copy(c.Response().Writer, res.Body)
				return err
			}
			// The authentication request was successful, inject headers and go to next middleware
			for _, hdr := range injectedHeaders {
				hdrValue := res.Header.Get(hdr)
				if hdrValue != "" {
					c.Request().Header.Set(hdr, hdrValue)
				}
			}
			return next(c)
		}
	}
}

// kgProjectsGraphStatusPathRewrite middleware
var kgProjectsGraphRewrites echo.MiddlewareFunc = middleware.RewriteWithConfig(middleware.RewriteConfig{
	RegexRules: map[*regexp.Regexp]string{
		regexp.MustCompile("^/api/projects/(.*?)/graph/webhooks(.*)"): "/projects/$1/webhooks$2",
		regexp.MustCompile("^/api/projects/(.*?)/graph/status(.*)"):   "/projects/$1/events/status$2",
	},
})

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

// printMsg prints the provided message when the middleware is accessed.
// Used only for troubleshooting and testing.
func printMsg(msg string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			slog.Info("reporting from middleware", "message", msg, "path", c.Request().URL.Path)
			return next(c)
		}
	}
}

// getMetricsServer creates a Prometheus server
func getMetricsServer(apiServer *echo.Echo, port int) *echo.Echo {
	metricsServer := echo.New()
	metricsServer.HideBanner = true
	// Skip the health endpoint
	urlSkipper := func(c echo.Context) bool {
		return strings.HasPrefix(c.Path(), "/revproxy/health")
	}
	prom := prometheus.NewPrometheus("gateway_revproxy", urlSkipper)
	prom.MetricsPath = "/"
	// Scrape metrics from Main Server
	apiServer.Use(prom.HandlerFunc)
	// Setup metrics endpoint at another server
	prom.SetMetricsPath(metricsServer)
	return metricsServer
}

// checkCoreServiceMetadataVersion checks if the requested path contains a valid
// and available metadata version and if not returns a 404, if the metadata version is
// available the request is let through
func checkCoreServiceMetadataVersion(coreSvcPaths []string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestedPath := c.Request().URL.Path
			match, err := regexp.MatchString(`^/api/renku/[0-9]+/.*$|^/api/renku/[0-9]+$`, requestedPath)
			if err != nil {
				return err
			}
			if !match {
				return next(c)
			}
			for _, path := range coreSvcPaths {
				if strings.HasPrefix(requestedPath, path) && path != "/api/renku" {
					return next(c)
				}
			}
			return echo.ErrNotFound
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

// uiServerUpstreamCoreLocation sets headers used by the UI server to determine where to route a specific Core service
// request. Allows us to position the UI server behind the gateway and still have the gateway "tell" the UI server
// where to route this Core service request. Used for only 1 specific endpoint that the UI server needs to cache so
// skipping the UI server on this endpoint is not possible.
func uiServerUpstreamCoreLocation(host string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			upstreamPath := *c.Request().URL
			upstreamPath.Host = ""
			upstreamPath.Scheme = ""
			upstreamPathStr := strings.TrimPrefix(upstreamPath.String(), "/ui-server")
			c.Request().Header.Set("Renku-Gateway-Upstream-Path", upstreamPathStr)
			c.Request().Header.Set("Renku-Gateway-Upstream-Host", host)
			return next(c)
		}
	}
}

// uiServerUpstreamKgLocation sets headers used by the UI server to determine where to route a specific KG request.
// Allows us to position the UI server behind the gateway and still have the gateway "tell" the UI server
// where to route this KG request. Used for only 1 specific endpoint on the KG that the UI server needs to cache
// so skipping the UI server on this endpoint is not possible.
func uiServerUpstreamKgLocation(host string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			upstreamPath := *c.Request().URL
			upstreamPath.Host = ""
			upstreamPath.Scheme = ""
			upstreamPathStr := strings.TrimPrefix(upstreamPath.String(), "/ui-server/api/kg")
			c.Request().Header.Set("Renku-Gateway-Upstream-Path", "/knowledge-graph"+upstreamPathStr)
			c.Request().Header.Set("Renku-Gateway-Upstream-Host", host)
			return next(c)
		}
	}
}

// uiServerInjectSessionId injects the session ID as a bearer token in the authorization header
// func uiServerInjectSessionId(next echo.HandlerFunc) echo.HandlerFunc {
// 	return func(c echo.Context) error {
// 		sessionID, ok := c.Get(commonconfig.SessionIDCtxKey).(string)
// 		if !ok {
// 			return gwerrors.ErrSessionParse
// 		}
// 		c.Request().Header.Set(http.CanonicalHeaderKey("authorization"), fmt.Sprintf("bearer %s", sessionID))
// 		return next(c)
// 	}
// }

// uiServerPathRewrite changes the incoming requests so that the UI server is used (as a second proxy) only for very
// specific endpoints (when absolutely necessary). For all other cases the gateway routes directly to the required
// Renku component and injects the proper credentials required by the specific component.
func UiServerPathRewrite() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path
			// For several endpoints below the gateway still cannot skip the UI server.
			if strings.HasPrefix(path, "/ui-server/api/projects") ||
				strings.HasPrefix(path, "/ui-server/api/renku/cache.files_upload") ||
				strings.HasPrefix(path, "/ui-server/api/kg/entities") ||
				strings.HasPrefix(path, "/ui-server/api/last-projects") ||
				strings.HasPrefix(path, "/ui-server/api/last-searches") {
				return next(c)
			}
			// For all other endpoints the gateway will fully bypass the UI server routing things directly to the proper
			// Renku component.
			if strings.HasPrefix(path, "/ui-server/api") {
				c.Request().URL.Path = strings.TrimPrefix(path, "/ui-server")
				c.Request().RequestURI = strings.TrimPrefix(c.Request().RequestURI, "/ui-server")
				c.Request().URL.RawPath = strings.TrimPrefix(c.Request().URL.RawPath, "/ui-server")
			}
			return next(c)
		}
	}
}

