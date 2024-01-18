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
		Balancer: middleware.NewRoundRobinBalancer([]*middleware.ProxyTarget{
			{
				Name: url.String(),
				URL:  url,
			}}),
	}
	return middleware.ProxyWithConfig(mwconfig)
}

func registerCoreSvcProxies(ctx context.Context, e *echo.Echo, revproxyConfig *config.RevproxyConfig, mwFuncs ...echo.MiddlewareFunc) {
	if len(revproxyConfig.RenkuServices.Core.ServicePaths) != len(revproxyConfig.RenkuServices.Core.ServiceNames) {
		e.Logger.Fatalf("Failed proxy setup for core service, number of paths (%d) and services (%d) provided does not match", len(revproxyConfig.RenkuServices.Core.ServicePaths), len(revproxyConfig.RenkuServices.Core.ServiceNames))
	}
	for i, service := range revproxyConfig.RenkuServices.Core.ServiceNames {
		path := revproxyConfig.RenkuServices.Core.ServicePaths[i]
		var coreBalancer middleware.ProxyBalancer
		imwFuncs := make([]echo.MiddlewareFunc, len(mwFuncs))
		copy(imwFuncs, mwFuncs)
		slog.Info("Setting up sticky sessions for %s with path %s", service, path)
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
