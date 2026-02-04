package main

import (
	"context"
	"log/slog"
	"os"

	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

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
		// Extract trace_id from context if available
		traceID := ""
		if span := sentryecho.GetSpanFromContext(c); span != nil {
			traceID = span.TraceID.String()
		}

		if v.Error == nil {
			attrs := []slog.Attr{
				slog.String("uri", v.URI),
				slog.Int("status", v.Status),
				slog.String("requestID", v.RequestID),
				slog.String("method", v.Method),
				slog.String("handler", v.RoutePath),
				slog.String("userAgent", v.UserAgent),
			}
			if traceID != "" {
				attrs = append(attrs, slog.String("sentryTraceID", traceID))
			}
			jsonLogger.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST", attrs...)
		} else {
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



var commonMiddlewares []echo.MiddlewareFunc = []echo.MiddlewareFunc{requestLogger}
