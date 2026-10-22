package utils

import (
	"errors"
	"log/slog"
	"net/textproto"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v5"
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

func FilterTransactions(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
	if event == nil {
		return nil
	}

	slog.Info("filter tx", "event", event, "hint", hint)

	renkuServiceHeaderValue := event.Request.Headers[textproto.CanonicalMIMEHeaderKey("renku-service")]
	slog.Info("filter tx: renkuServiceHeaderValue", "renkuServiceHeaderValue", renkuServiceHeaderValue)
	if strings.EqualFold(renkuServiceHeaderValue, "renku-ui-server") {
		// This request is coming from renku-ui-server, reduce sample rate
		trace := event.Contexts["trace"]
		slog.Info("filter tx: trace", "trace", trace)
	}

	return nil
}
