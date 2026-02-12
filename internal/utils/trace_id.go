package utils

import (
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo/v4"
)

func GetTraceID(c echo.Context) string {
	if span := sentryecho.GetSpanFromContext(c); span != nil {
		return span.TraceID.String()
	}
	return ""
}
