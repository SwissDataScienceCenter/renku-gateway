package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/utils"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
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
		traceID := utils.GetTraceID(c)
		if v.Error == nil {
			jsonLogger.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST",
				slog.String("uri", v.URI),
				slog.Int("status", v.Status),
				slog.String("requestID", v.RequestID),
				slog.String("method", v.Method),
				slog.String("handler", v.RoutePath),
				slog.String("userAgent", v.UserAgent),
				slog.String("traceID", traceID),
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
				slog.String("traceID", traceID),
			)
		}
		return nil
	},
})

// sentryHeaderInjector ensures that OpenTelemetry and Sentry trace headers are attached to the outgoing request.
func sentryHeaderInjector(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		propagator := otel.GetTextMapPropagator()
		propagator.Inject(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))
		return next(c)
	}
}

var commonMiddlewares []echo.MiddlewareFunc = []echo.MiddlewareFunc{sentryHeaderInjector, requestLogger}
