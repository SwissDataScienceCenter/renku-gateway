package utils

import (
	"net/http"

	"github.com/getsentry/sentry-go"
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo/v4"
)

func GetTraceID(c echo.Context) string {
	if span := sentryecho.GetSpanFromContext(c); span != nil {
		return span.TraceID.String()
	}
	return ""
}

// GetTraceIDFromHTTPRequest extracts the trace ID from an http.Request.
func GetTraceIDFromHTTPRequest(r *http.Request) string {
	if hub := sentry.GetHubFromContext(r.Context()); hub != nil {
		return hub.GetTraceparent()
	}
	return ""
}
