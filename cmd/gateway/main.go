package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/authentication"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/db"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/login"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/revproxy"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/tokenstore"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/views"
	"github.com/getsentry/sentry-go"
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

func main() {
	// Logging setup
	slog.SetDefault(jsonLogger)
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
	// Set log level to "debug" if activated
	if gwConfig.DebugMode {
		logLevel.Set(slog.LevelDebug)
	}
	// Setup
	e := echo.New()
	e.Pre(middleware.RequestID(), middleware.RemoveTrailingSlash(), revproxy.UiServerPathRewrite())
	e.Use(middleware.Recover())
	// The banner and the port do not respect the logger formatting we set below so we remove them
	// the port will be logged further down when the server starts.
	e.HideBanner = true
	e.HidePort = true
	// Setup template renderer
	tr, err := views.NewTemplateRenderer()
	if err != nil {
		slog.Error("Template renderer initialization failed", "error", err)
		os.Exit(1)
	}
	tr.Register(e)
	// Health check
	e.GET("/health", func(c echo.Context) error {
		// TODO: maybe implement a real health check
		return c.NoContent(http.StatusOK)
	})
	// Version endpoint
	buildInfo, ok := debug.ReadBuildInfo()
	version := ""
	if ok && buildInfo != nil {
		version = buildInfo.Main.Version
	}
	e.GET("/version", func(c echo.Context) error {
		return c.String(http.StatusOK, version)
	})
	// Initialize the db adapters
	dbOptions := []db.RedisAdapterOption{db.WithRedisConfig(gwConfig.Redis)}
	if gwConfig.Login.TokenEncryption.Enabled && gwConfig.Login.TokenEncryption.SecretKey != "" {
		slog.Info("redis encryption is enabled")
		dbOptions = append(dbOptions, db.WithEcryption(string(gwConfig.Login.TokenEncryption.SecretKey)))
	}
	dbAdapter, err := db.NewRedisAdapter(dbOptions...)
	if err != nil {
		slog.Error("DB adapter initialization failed", "error", err)
		os.Exit(1)
	}
	// Initialize the token store
	tokenStore, err := tokenstore.NewTokenStore(
		tokenstore.WithExpiryMargin(time.Duration(3)*time.Minute),
		tokenstore.WithConfig(gwConfig.Login),
		tokenstore.WithTokenRepository(dbAdapter),
	)
	if err != nil {
		slog.Error("token store initialization failed", "error", err)
		os.Exit(1)
	}
	// Create authenticator
	authenticator, err := authentication.NewAuthenticator(authentication.WithConfig(gwConfig.Sessions.AuthorizationVerifiers))
	if err != nil {
		slog.Error("failed to initialize authenticator", "error", err)
		os.Exit(1)
	}
	// Create session store
	sessionStore, err := sessions.NewSessionStore(
		sessions.WithAuthenticator(authenticator),
		sessions.WithSessionRepository(dbAdapter),
		sessions.WithTokenStore(tokenStore),
		sessions.WithConfig(gwConfig.Sessions),
	)
	if err != nil {
		slog.Error("failed to initialize sessions", "error", err)
		os.Exit(1)
	}
	// Add the session store to the common middlewares
	gwMiddlewares := append(commonMiddlewares, sessionStore.Middleware())
	// Initialize the reverse proxy
	revproxy, err := revproxy.NewServer(revproxy.WithConfig(gwConfig.Revproxy), revproxy.WithSessionStore(sessionStore))
	if err != nil {
		slog.Error("revproxy handlers initialization failed", "error", err)
		os.Exit(1)
	}
	revproxy.RegisterHandlers(e, gwMiddlewares...)
	// Initialize login server
	loginServer, err := login.NewLoginServer(login.WithConfig(gwConfig.Login), login.WithSessionStore(sessionStore), login.WithTokenStore(tokenStore))
	if err != nil {
		slog.Error("login handlers initialization failed", "error", err)
		os.Exit(1)
	}
	loginServer.RegisterHandlers(e, gwMiddlewares...)
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
			Dsn:              string(gwConfig.Monitoring.Sentry.Dsn),
			TracesSampleRate: gwConfig.Monitoring.Sentry.SampleRate,
			Environment:      gwConfig.Monitoring.Sentry.Environment,
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
