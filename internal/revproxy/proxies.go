package revproxy

import (
	"log/slog"
	"net/url"
	"os"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/utils"
	"github.com/go-extras/errx"
	"github.com/go-extras/errx/stacktrace"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var ErrProxy = errx.NewSentinel("proxy error")

// proxyFromURL middleware creates a proxy that forwards requests to the specified URL
func proxyFromURL(url *url.URL) echo.MiddlewareFunc {
	if url == nil {
		slog.Error("cannot create a proxy from a nil URL")
		os.Exit(1)
	}
	mwConfig := middleware.ProxyConfig{
		// the skipper is used to log only
		Skipper: func(c echo.Context) bool {
			slog.Info("PROXY", "requestID", utils.GetRequestID(c), "destination", url.String())
			return false
		},
		Balancer: middleware.NewRoundRobinBalancer([]*middleware.ProxyTarget{
			{
				Name: url.String(),
				URL:  url,
			}}),
		ErrorHandler: func(c echo.Context, err error) error {
			return errx.Wrap("Unhandled error", err, errx.Attrs("proxy_url", url, "echo_context", c), ErrProxy, stacktrace.Here())
		},
	}
	return middleware.ProxyWithConfig(mwConfig)
}
