package revproxy

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/stickysessions"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// proxyFromURL middleware creates a proxy that forwards requests to the specified URL
func proxyFromURL(url *url.URL) echo.MiddlewareFunc {
	if url == nil {
		slog.Error("cannot create a proxy from a nil URL")
		os.Exit(1)
	}
	mwconfig := middleware.ProxyConfig{
		// the skipper is used to log only
		Skipper: func(c echo.Context) bool {
			slog.Info("PROXY", "requestID", c.Request().Header.Get("X-Request-ID"), "destination", url.String())
			return false
		},
		Balancer: middleware.NewRoundRobinBalancer([]*middleware.ProxyTarget{
			{
				Name: url.String(),
				URL:  url,
			}}),
	}
	return middleware.ProxyWithConfig(mwconfig)
}

// registerCoreSvcProxies creates and registers all proxies for the core service. The core service is special
// because it runs multiple API versions of itself at the same time and the gateway has to route between them.
// In addition, and even more importantly, the core service requires sticky sessions between different pods of
// a deployment that runs the same version of the API. So we have to implement our own custom load balancer that
// can distinguish between different pods that sit behind a K8s service and consistently send requests to the same pod.
func registerCoreSvcProxies(ctx context.Context, e *echo.Echo, revproxyConfig *config.RevproxyConfig, mwFuncs ...echo.MiddlewareFunc) {
	if len(revproxyConfig.RenkuServices.Core.ServicePaths) != len(revproxyConfig.RenkuServices.Core.ServiceNames) {
		e.Logger.Fatalf("Failed proxy setup for core service, number of paths (%d) and services (%d) provided does not match", len(revproxyConfig.RenkuServices.Core.ServicePaths), len(revproxyConfig.RenkuServices.Core.ServiceNames))
	}
	for i, service := range revproxyConfig.RenkuServices.Core.ServiceNames {
		path := revproxyConfig.RenkuServices.Core.ServicePaths[i]
		var coreBalancer middleware.ProxyBalancer
		imwFuncs := make([]echo.MiddlewareFunc, len(mwFuncs))
		copy(imwFuncs, mwFuncs)
		slog.Info("STICKY SESSIONS SETUP", "service", service, "path", path)
		if revproxyConfig.RenkuServices.Core.Sticky {
			cookieName := fmt.Sprintf("reverse-proxy-sticky-session-%s", service)
			coreBalancer = stickysessions.NewStickySessionBalancer(ctx, service, revproxyConfig.K8sNamespace, "http", "/", cookieName)
		} else {
			url, err := url.Parse(service)
			if err != nil {
				e.Logger.Fatal(err)
			}
			coreBalancer = middleware.NewRandomBalancer([]*middleware.ProxyTarget{{URL: url}})
		}
		coreStickSessionsProxy := middleware.Proxy(coreBalancer)
		imwFuncs = append(imwFuncs, coreStickSessionsProxy)
		e.Group(path, imwFuncs...)
	}
}
