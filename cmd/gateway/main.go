package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/login"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/revproxy"
	"github.com/getsentry/sentry-go"
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

func main() {
	// Setup
	e := echo.New()
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Recover(), middleware.RequestID())
	// The banner and the port do not respect the logger formatting we set below so we remove them
	// the port will be logged further down when the server starts.
	e.HideBanner = true
	e.HidePort = true
	// Logging setup
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
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
				logger.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
				)
			} else {
				logger.LogAttrs(context.Background(), slog.LevelError, "REQUEST_ERROR",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.String("error", v.Error.Error()),
				)
			}
			return nil
		},
	}))
	// Load configuration
	ch := config.NewConfigHandler()
	gwConfig, err := ch.Config()
	if err != nil {
		slog.Error("loading the configuration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("loaded config", "config", gwConfig)
	err = gwConfig.Validate()
	if err != nil {
		slog.Error("the config validation failed", "error", err)
		os.Exit(1)
	}
	ch.Watch()
	var restart bool = false
	ch.HandleChanges(func(c config.Config, err error) {
		// when the config changes we flip the restart flag to true and cause the health endpoint to
		// fail which will cause K8s to kill the pod
		slog.Info("config file changed, making health check return status 500")
		restart = true
	})
	// Health check
	e.GET("/health", func(c echo.Context) error {
		if restart {
			slog.Warn("responding with error status to the health endpoint, server restart is imminent")
			return c.NoContent(http.StatusInternalServerError)
		}
		return c.NoContent(http.StatusOK)
	})
	// Version endpoint
	version := os.Getenv("VERSION")
	e.GET("/version", func(c echo.Context) error {
		return c.String(http.StatusOK, version)
	})
	// Initialize the reverse proxy
	revproxy := revproxy.NewServer(&gwConfig.Revproxy)
	revproxy.RegisterHandlers(e)
	// Initialize login server
	loginServer, err := login.NewLoginServer(login.WithConfig(gwConfig.Login), login.WithDBConfig(gwConfig.Redis))
	if err != nil {
		slog.Error("login handlers initialization failed", "error", err)
		os.Exit(1)
	}
	loginServer.RegisterHandlers(e)
	// Rate limiting
	if gwConfig.Server.RateLimits.Enabled {
		e.Use(middleware.RateLimiter(
			middleware.NewRateLimiterMemoryStoreWithConfig(
				middleware.RateLimiterMemoryStoreConfig{
					Rate:      rate.Limit(gwConfig.Server.RateLimits.Rate),
					Burst:     gwConfig.Server.RateLimits.Burst,
					ExpiresIn: 3 * time.Minute,
				}),
		),
		)
	}
	// CORS
	if len(gwConfig.Server.AllowOrigin) > 0 {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{AllowOrigins: gwConfig.Server.AllowOrigin}))
	}
	// Sentry
	if gwConfig.Monitoring.Sentry.Enabled {
		err := sentry.Init(sentry.ClientOptions{
			Dsn: string(gwConfig.Monitoring.Sentry.Dsn),
			TracesSampleRate: gwConfig.Monitoring.Sentry.SampleRate, 
			Environment: gwConfig.Monitoring.Sentry.Environment,	
		})
		if err != nil {
			slog.Error("sentry initialization failed", "error", err)
		}
		e.Use(sentryecho.New(sentryecho.Options{}))
	}
	// Prometheus
	if gwConfig.Monitoring.Prometheus.Enabled {
		e.Use(echoprometheus.NewMiddleware("gateway"))
		go func() {
			metrics := echo.New()
			metrics.HideBanner = true
			metrics.HidePort = true
			metrics.GET("/metrics", echoprometheus.NewHandler())
			err := metrics.Start(fmt.Sprintf(":%d", gwConfig.Monitoring.Prometheus.Port))
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Error("prometheus server failed to start", "error", err)
				os.Exit(1)
			}
		}()
	}
	// Start server
	address := fmt.Sprintf("%s:%d", gwConfig.Server.Host, gwConfig.Server.Port)
	slog.Info("starting the server on address " + address)
	go func() {
		err := e.Start(address)
		if err != nil && err != http.ErrServerClosed {
			slog.Error("shutting down the server gracefuly failed", "error", err)
			os.Exit(1)
		}
	}()
	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	slog.Info("received signal to shut down the server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		slog.Error("shutting down the server gracefully failed", "error", err)
		os.Exit(1)
	}
}

