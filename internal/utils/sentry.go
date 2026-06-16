package utils

import (
	"errors"
	"strings"

	"github.com/labstack/echo/v4"
)

// SendErrorToSentry returns true if the given error should be sent to Sentry
func SendErrorToSentry(err error) bool {
	if err == nil {
		return false
	}
	// Do not report 404 errors
	if errors.Is(err, echo.ErrNotFound) {
		return false
	}
	// Do not report "proxy raw, copy body error"
	if strings.HasPrefix(err.Error(), "proxy raw, copy body error=") {
		return false
	}
	return true
}
