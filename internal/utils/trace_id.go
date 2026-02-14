package utils

import (
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/trace"
)

// GetTraceID extracts the trace ID from the request context. First looks for OpenTelemetry trace ID and then for Sentry
// trace ID if OpenTelemetry is not available.
func GetTraceID(c echo.Context) string {
	if span := trace.SpanFromContext(c.Request().Context()); span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	if span := sentryecho.GetSpanFromContext(c); span != nil {
		return span.TraceID.String()
	}
	return ""
}
