package main

import (
	"time"

	"github.com/labstack/echo/v4"
)

// NoCaching sets headers in responses that prevent caching by the browser.
// Taken from oauth2 proxy.
func NoCaching(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		var noCacheHeaders = map[string]string{
			"Expires":         time.Unix(0, 0).Format(time.RFC1123),
			"Cache-Control":   "no-cache, no-store, must-revalidate, max-age=0",
			"X-Accel-Expires": "0",
		}
		for k, v := range noCacheHeaders {
			c.Response().Header().Set(k, v)
		}
		return next(c)
	}
}
