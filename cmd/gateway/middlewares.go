package main

import (
	"context"
	"log/slog"
	"os"

	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const sentryTraceIDKey = "sentry_trace_id"

var logLevel *slog.LevelVar = new(slog.LevelVar)
var jsonLogger *slog.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
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
			// jsonLogger.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST",
			// 	slog.String("uri", v.URI),
			// 	slog.Int("status", v.Status),
			// 	slog.String("requestID", v.RequestID),
			// 	slog.String("method", v.Method),
			// 	slog.String("handler", v.RoutePath),
			// 	slog.String("userAgent", v.UserAgent),
			// )
		} else {
			// Extract trace_id from context if available
			traceID := "MISSING"
			if tid, ok := c.Get(sentryTraceIDKey).(string); ok {
				traceID = tid
			}

			attrs := []slog.Attr{
				slog.String("uri", v.URI),
				slog.Int("status", v.Status),
				slog.String("error", v.Error.Error()),
				slog.String("requestID", v.RequestID),
				slog.String("method", v.Method),
				slog.String("handler", v.RoutePath),
				slog.String("userAgent", v.UserAgent),
			}
			if traceID != "" {
				attrs = append(attrs, slog.String("sentryTraceID", traceID))
			}
			jsonLogger.LogAttrs(context.Background(), slog.LevelError, "REQUEST_ERROR", attrs...)
		}
		return nil
	},
})

// sentryTraceIDExtractor extracts the Sentry trace ID and stores it on the context for later use
var sentryTraceIDExtractor echo.MiddlewareFunc = func(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if span := sentryecho.GetSpanFromContext(c); span != nil {
			traceID := span.TraceID.String()
			c.Set(sentryTraceIDKey, traceID)
		}
		return next(c)
	}
}

var commonMiddlewares []echo.MiddlewareFunc = []echo.MiddlewareFunc{requestLogger}
