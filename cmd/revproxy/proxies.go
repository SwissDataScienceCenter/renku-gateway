package main

import (
	"net/url"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// proxyFromURL middleware creates a proxy that forwards requests to the specified URL
func proxyFromURL(url *url.URL) echo.MiddlewareFunc {
	config := middleware.ProxyConfig{
		Balancer: middleware.NewRoundRobinBalancer([]*middleware.ProxyTarget{
			{
				Name: url.String(),
				URL:  url,
			}}),
	}
	return middleware.ProxyWithConfig(config)
}
