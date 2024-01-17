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
