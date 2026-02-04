package revproxy

import (
	"log/slog"
	"net/http"
	"net/url"
	"os"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/utils"
	"github.com/getsentry/sentry-go"
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// A custom RoundTripper that adds Sentry trace headers to outgoing requests
type sentryPropagationTransport struct {
	next http.RoundTripper
}

func (t *sentryPropagationTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	slog.Info("ROUND-TRIP ========================= BEFORE adding headers to outgoing request",
		"url", request.URL.String(),
		"all_headers", request.Header,
		"has_context", request.Context() != nil)

	// TODO: Do we need to get the hub?
	if hub := sentry.GetHubFromContext(request.Context()); hub != nil {
		if span := sentry.TransactionFromContext(request.Context()); span != nil {
			sentryTraceHeader := span.ToSentryTrace()
			baggageHeader := span.ToBaggage()

			// TODO: Log headers to see if they already contain Sentry headers
			slog.Info("ROUND-TRIP ========================= adding headers to outgoing request",
				"sentry-trace", sentryTraceHeader,
				"baggage", baggageHeader,
				"url", request.URL.String())

			request.Header.Set("sentry-trace", sentryTraceHeader)
			if baggageHeader != "" {
				request.Header.Set("baggage", baggageHeader)
			}

			slog.Info("ROUND-TRIP ========================= AFTER adding headers to outgoing request", "all_headers", request.Header)
		} else {
			slog.Info("No Sentry span in request context", "url", request.URL.String())
		}
	} else {
		slog.Info("No Sentry hub in request context", "url", request.URL.String())
	}

	// TODO: Is it correct to call the next transport in the chain
	return t.next.RoundTrip(request)
}

// proxyFromURL middleware creates a proxy that forwards requests to the specified URL
func proxyFromURL(url *url.URL) echo.MiddlewareFunc {
	if url == nil {
		slog.Error("cannot create a proxy from a nil URL")
		os.Exit(1)
	}
	config := middleware.ProxyConfig{
		// the skipper is used to log only
		Skipper: func(c echo.Context) bool {
			slog.Info("PROXY ========================= incoming request headers", "all_headers", c.Request().Header)

			traceID := "MISSING"
			if span := sentryecho.GetSpanFromContext(c); span != nil {
				traceID = span.TraceID.String()
			}
			if traceID != "" {
				slog.Info("PROXY", "requestID", utils.GetRequestID(c), "destination", url.String(), "sentryTraceID", traceID)
			} else {
				slog.Info("PROXY", "requestID", utils.GetRequestID(c), "destination", url.String())
			}
			return false
		},
		Balancer: middleware.NewRoundRobinBalancer([]*middleware.ProxyTarget{
			{
				Name: url.String(),
				URL:  url,
			}}),
		// Use custom transport to add Sentry headers to outgoing requests
		Transport: &sentryPropagationTransport{
			next: http.DefaultTransport,
		},
	}
	return middleware.ProxyWithConfig(config)
}
