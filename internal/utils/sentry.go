package utils

import (
	"errors"
	"log/slog"

	"github.com/labstack/echo/v4"
)

// SendErrorToSentry returns true if the given error should be sent to Sentry
func SendErrorToSentry(err error) bool {
	// Do not report 404 errors
	if errors.Is(err, echo.ErrNotFound) {
		slog.Error("FILTERING 404 ERROR")
		return false
	}
	return true
}
