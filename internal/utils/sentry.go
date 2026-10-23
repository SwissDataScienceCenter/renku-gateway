package utils

import (
	"context"
	"errors"
	"log/slog"
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
	reducedSampleRate := sampleRate / 1_000

	return func(ctx sentry.SamplingContext) float64 {
		span := ctx.Span
		if span == nil {
			return sampleRate
		}

		data, ok := span.Context().Value(samplerContextKey{}).(*samplerContextData)
		slog.Info("TracesSampler: DEBUG", "ok", ok, "data", data, "data.c", (data.c != nil))
		if ok && data != nil && data.c != nil && data.c.Request() != nil {
			r := data.c.Request()
			slog.Info("TracesSampler: DEBUG", "r.URL", r.URL.String(), "r.Header", r.Header, "test", r.Header.Values("renku-service"))
			for _, val := range r.Header.Values("renku-service") {
				slog.Info("TracesSampler: DEBUG2", "val", val, "=", val == "renku-ui-server", "=fold", strings.EqualFold(val, "renku-ui-server"))
				if strings.EqualFold(val, "renku-ui-server") {
					// This request is coming from renku-ui-server, reduce sample rate
					slog.Info("TracesSampler: changing sample rate", "sampleRate", sampleRate, "reducedSampleRate", reducedSampleRate,
						"Parent.Sampled", ctx.Parent.Sampled,
						"ParentSampled", ctx.ParentSampled,
					)
					return reducedSampleRate
				}
			}
		}

		slog.Info("TracesSampler: INFO",
			"Parent", ctx.Parent,
			"ParentSampled", ctx.ParentSampled,
		)

		// Inherit decision from a local parent span (same process).
		if ctx.Parent != nil {
			if ctx.Parent.Sampled == sentry.SampledTrue {
				return 1.0
			}
			return 0.0
		}
		// Inherit decision from a remote parent span (upstream service).
		if ctx.ParentSampled != sentry.SampledUndefined {
			if ctx.ParentSampled == sentry.SampledTrue {
				return 1.0
			}
			return 0.0
		}

		return sampleRate
	}
}
