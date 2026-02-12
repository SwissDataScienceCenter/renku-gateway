package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/getsentry/sentry-go"
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var logLevel *slog.LevelVar = new(slog.LevelVar)
var jsonLogger *slog.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

// sentryHeaderInjector ensures that the Trace ID is attached to the outgoing request.
func sentryHeaderInjector(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		hub := sentryecho.GetHubFromContext(c)
		if hub != nil {
			c.Request().Header.Set(sentry.SentryTraceHeader, hub.GetTraceparent())
			c.Request().Header.Set(sentry.SentryBaggageHeader, hub.GetBaggage())
		}

		// if span := sentryecho.GetSpanFromContext(c); span != nil {
		// 	sentryTraceHeader := span.ToSentryTrace()
		// 	baggageHeader := span.ToBaggage()
		// 	c.Request().Header.Set(sentry.SentryTraceHeader, sentryTraceHeader)
		// 	if baggageHeader != "" {
		// 		c.Request().Header.Set(sentry.SentryBaggageHeader, baggageHeader)
		// 	}
		// }
		return next(c)
	}
}

var requestLogger echo.MiddlewareFunc = middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
	LogStatus:    true,
	LogURI:       true,
	LogError:     true,
	LogRequestID: true,
	LogRoutePath: true, // logs the handler path in the server that matched the request path
	LogMethod:    true,
	LogUserAgent: true,
	HandleError:  true, // forwards error to the global error handler, so it can decide appropriate status code
	LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
		if v.Error == nil {
			jsonLogger.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST",
				slog.String("uri", v.URI),
				slog.Int("status", v.Status),
				slog.String("requestID", v.RequestID),
				slog.String("method", v.Method),
				slog.String("handler", v.RoutePath),
				slog.String("userAgent", v.UserAgent),
			)
		} else {
			jsonLogger.LogAttrs(context.Background(), slog.LevelError, "REQUEST_ERROR",
				slog.String("uri", v.URI),
				slog.Int("status", v.Status),
				slog.String("error", v.Error.Error()),
				slog.String("requestID", v.RequestID),
				slog.String("method", v.Method),
				slog.String("handler", v.RoutePath),
				slog.String("userAgent", v.UserAgent),
			)
		}
		return nil
	},
})

var commonMiddlewares []echo.MiddlewareFunc = []echo.MiddlewareFunc{sentryHeaderInjector, requestLogger}
