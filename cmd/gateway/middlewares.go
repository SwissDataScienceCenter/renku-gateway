package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var jsonLogger *slog.Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
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
var commonMiddlewares []echo.MiddlewareFunc = []echo.MiddlewareFunc{requestLogger}

