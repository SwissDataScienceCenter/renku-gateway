package revproxy

import (
	"log/slog"
	"net/url"
	"os"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/utils"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// proxyFromURL middleware creates a proxy that forwards requests to the specified URL
func proxyFromURL(url *url.URL) echo.MiddlewareFunc {
	if url == nil {
		slog.Error("cannot create a proxy from a nil URL")
		os.Exit(1)
	}
	mwConfig := middleware.ProxyConfig{
		// the skipper is used to log only
		Skipper: func(c echo.Context) bool {
			slog.Info("PROXY", "requestID", utils.GetRequestID(c), "traceID", utils.GetTraceID(c), "destination", url.String())
			return false
		},
		Balancer: middleware.NewRoundRobinBalancer([]*middleware.ProxyTarget{
			{
				Name: url.String(),
				URL:  url,
			}}),
	}
	return middleware.ProxyWithConfig(mwConfig)
}
