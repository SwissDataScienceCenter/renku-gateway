package utils

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v5"
)

// type myContextKey struct{}
// type myContextData struct {
// 	// Store information for custom TracesSampler.
// 	request *http.Request
// 	// ...
// }

type samplerContextKey struct{}
type samplerContextData struct {
	c *echo.Context
}

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

// func(next HandlerFunc) HandlerFunc
func SamplerMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		r := c.Request()
		data := &samplerContextData{c}
		ctx := context.WithValue(r.Context(), samplerContextKey{}, data)
		r = r.WithContext(ctx)
		c.SetRequest(r)
		return next(c)
	}
}

func TracesSampler(sampleRate float64) sentry.TracesSampler {
	return func(ctx sentry.SamplingContext) float64 {
		// slog.Info("TracesSampler", "ctx", ctx)
		span := ctx.Span
		if span == nil {
			return sampleRate
		}

		data, ok := span.Context().Value(samplerContextKey{}).(*samplerContextData)
		var r *http.Request = nil
		if ok {
			r = data.c.Request()
		}
		slog.Info("TracesSampler", "data", data, "ok", ok, "r", r)

		// // spanCtx := span.Context()
		// hub := sentry.GetHubFromContext(span.Context())
		// // sentry.CurrentHub().GetBaggage()
		// slog.Info("TracesSampler", "ctx", ctx, "spanCtx", span.Context(), "hub", hub)

		return sampleRate
	}
}

// func FilterTransactions(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
// 	if event == nil {
// 		return nil
// 	}

// 	slog.Info("filter tx", "event", event, "hint", hint)

// 	renkuServiceHeaderValue := event.Request.Headers[textproto.CanonicalMIMEHeaderKey("renku-service")]
// 	slog.Info("filter tx: renkuServiceHeaderValue", "renkuServiceHeaderValue", renkuServiceHeaderValue)
// 	if strings.EqualFold(renkuServiceHeaderValue, "renku-ui-server") {
// 		// This request is coming from renku-ui-server, reduce sample rate
// 		dsc := event.GetDynamicSamplingContext()
// 		slog.Info("filter tx: dsc", "dsc", dsc)
// 		oldSampleRate, err := strconv.ParseFloat(dsc["sample_rate"], 64)
// 		if err != nil {
// 			return nil // TODO: return event
// 		}
// 		sampleRate := oldSampleRate / 1_000

// 	}

// 	return nil
// }
