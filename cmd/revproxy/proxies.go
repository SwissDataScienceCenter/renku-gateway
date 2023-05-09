package main

import (
	"net/url"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/stickysessions"
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

func registerCoreSvcProxies(e *echo.Echo, paths []string, services []string, namespace string, mwFuncs ...echo.MiddlewareFunc) {
	if len(paths) != len(services) {
		e.Logger.Fatalf("Failed proxy setup for core service, number of paths (%d) and services (%d) provided does not match", len(paths), len(services))
	}
	for i, service := range services {
		path := paths[i]
		e.Logger.Printf("Setting up sticky sessions for %s with path %s", service, path)
		coreBalancer := stickysessions.NewStickySessionBalancer(service, namespace, "http", path)
		coreStickSessionsProxy := middleware.Proxy(coreBalancer)
		imwFuncs := make([]echo.MiddlewareFunc, len(mwFuncs))
		copy(imwFuncs, mwFuncs)
		imwFuncs = append(imwFuncs, coreStickSessionsProxy)
		e.Group(path, imwFuncs...)
	}
}
