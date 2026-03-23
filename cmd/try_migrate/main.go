package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	_ "github.com/SwissDataScienceCenter/renku-gateway/internal/pg/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/pressly/goose/v3"
)

// Try: we can do something like this at the start of the gateway process to run migrations
func main() {
	slog.SetDefault(jsonLogger)
	ch := config.NewConfigHandler()
	gwConfig, err := ch.Config()
	if err != nil {
		slog.Error("loading the configuration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("loaded config", "config", gwConfig)
	err = gwConfig.Postgres.Validate()
	if err != nil {
		slog.Error("the config validation failed", "error", err)
		os.Exit(1)
	}

	dbURL, err := getPostgresURL(gwConfig.Postgres)
	if err != nil {
		slog.Error("could not form postgres database URL", "error", err)
		os.Exit(1)
	}

	connConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		slog.Error("could not create database connection", "error", err)
		os.Exit(1)
	}

	dbpool, err := pgxpool.NewWithConfig(context.Background(), connConfig)
	if err != nil {
		slog.Error("could not create database connection pool", "error", err)
		os.Exit(1)
	}
	defer dbpool.Close()

	connConfig.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger:   &traceLogger{},
		LogLevel: tracelog.LogLevelTrace,
	}

	db, err := goose.OpenDBWithDriver("pgx", stdlib.RegisterConnConfig(connConfig.ConnConfig))
	if err != nil {
		slog.Error("goose: unable to connect to database", "error", err)
		os.Exit(1)
	}

	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("goose: failed to close DB", "error", err)
			os.Exit(1)
		}
	}()

	err = goose.UpContext(context.Background(), db, "internal/pg/migrations")
	if err != nil {
		slog.Error("goose: failed to run migrations", "error", err)
		os.Exit(1)
	}
}

func getPostgresURL(c config.PostgresConfig) (postgresURL string, err error) {
	url, err := url.Parse(fmt.Sprintf("postgres://%s:%s@%s:5432/%s", c.Username, string(c.Password), c.Host, c.Database))
	if err != nil {
		return postgresURL, err
	}
	return url.String(), nil
}

var logLevel *slog.LevelVar = new(slog.LevelVar)
var jsonLogger *slog.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

type traceLogger struct{}

func (tl *traceLogger) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
	attrs := []slog.Attr{{Key: "Lvl", Value: slog.StringValue(level.String())}}
	for key, value := range data {
		attrs = append(attrs, slog.Attr{Key: key, Value: slog.AnyValue(value)})
	}

	slog.Default().LogAttrs(ctx, slog.LevelError, msg, attrs...)
}
