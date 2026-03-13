package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"

	_ "github.com/SwissDataScienceCenter/renku-gateway/internal/pg/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/pressly/goose/v3"
)

var (
	flags = flag.NewFlagSet("goose", flag.ExitOnError)
	dir   = flags.String("dir", "internal/pg/migrations", "directory with migration files")
)

// Try: we can do something like this at the start of the gateway process to run migrations
func main() {
	dbURL, err := getPostgresURL()
	if err != nil {
		log.Fatalf("goose: could not form postgres database URL: %v\n", err)
	}

	connConfig, err := pgx.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("goose: could not create database connection: %v\n", err)
	}
	connConfig.Tracer = &tracelog.TraceLog{
		Logger:   &traceLogger{},
		LogLevel: tracelog.LogLevelTrace,
	}

	db, err := goose.OpenDBWithDriver("pgx", stdlib.RegisterConnConfig(connConfig))
	if err != nil {
		log.Fatalf("goose: unable to connect to database: %v", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Fatalf("goose: failed to close DB: %v", err)
		}
	}()

	err = goose.UpContext(context.Background(), db, *dir)
	if err != nil {
		log.Fatalf("goose: failed to run migrations: %v", err)
	}
}

func getPostgresURL() (postgresURL string, err error) {
	// TODO: use gateway config
	dbName := os.Getenv(("DB_NAME"))
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	host := os.Getenv("DB_HOST")
	url, err := url.Parse(fmt.Sprintf("postgres://%s:%s@%s:5432/%s", user, password, host, dbName))
	if err != nil {
		return postgresURL, err
	}
	return url.String(), nil
}

type traceLogger struct{}

func (tl *traceLogger) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
	attrs := []slog.Attr{{Key: "Lvl", Value: slog.StringValue(level.String())}}
	for key, value := range data {
		attrs = append(attrs, slog.Attr{Key: key, Value: slog.AnyValue(value)})
	}

	slog.Default().LogAttrs(ctx, slog.LevelError, msg, attrs...)
}
