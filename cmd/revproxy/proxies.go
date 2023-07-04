package main

import (
	"context"
	"fmt"
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

func registerCoreSvcProxies(ctx context.Context, e *echo.Echo, config revProxyConfig, mwFuncs ...echo.MiddlewareFunc) {
	if len(config.RenkuServices.CoreServicePaths) != len(config.RenkuServices.CoreServiceNames) {
		e.Logger.Fatalf("Failed proxy setup for core service, number of paths (%d) and services (%d) provided does not match", len(config.RenkuServices.CoreServicePaths), len(config.RenkuServices.CoreServiceNames))
	}
	for i, service := range config.RenkuServices.CoreServiceNames {
		path := config.RenkuServices.CoreServicePaths[i]
		var coreBalancer middleware.ProxyBalancer
		imwFuncs := make([]echo.MiddlewareFunc, len(mwFuncs))
		copy(imwFuncs, mwFuncs)
		e.Logger.Printf("Setting up sticky sessions for %s with path %s", service, path)
		if config.Debug {
			url, err := url.Parse(service)
			if err != nil {
				e.Logger.Fatal(err)
			}
			coreBalancer = middleware.NewRandomBalancer([]*middleware.ProxyTarget{{URL: url}})
		} else {
			cookieName := fmt.Sprintf("reverse-proxy-sticky-session-%s", service)			
			coreBalancer = stickysessions.NewStickySessionBalancer(ctx, service, config.Namespace, "http", "/", cookieName)
		}
		coreStickSessionsProxy := middleware.Proxy(coreBalancer)
		imwFuncs = append(imwFuncs, coreStickSessionsProxy)
		e.Group(path, imwFuncs...)
	}
}
