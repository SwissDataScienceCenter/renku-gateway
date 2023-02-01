package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"

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
		regexp.MustCompile("^/api/projects/(.*?)/graph/webhooks/(.*)"): "/projects/$1/webhooks/$2",
		regexp.MustCompile("^/api/projects/(.*?)/graph/status/(.*)"):   "/projects/$1/events/status/$2",
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
			regexp.MustCompile(fmt.Sprintf("^%s/(.*)", prefix)): "/$1",
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
			log.Printf("Printing msg '%s' at path %s\n", msg, c.Request().URL.Path)
			return next(c)
		}
	}
}
