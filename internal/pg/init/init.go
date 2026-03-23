package init

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	_ "github.com/SwissDataScienceCenter/renku-gateway/internal/pg/migrations"
	pgutils "github.com/SwissDataScienceCenter/renku-gateway/internal/pg/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func RunPostgresMigrationsWithTimeout(ctx context.Context, config config.PostgresConfig, logLevel slog.Level, timeout time.Duration) error {
	childCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return RunPostgresMigrations(childCtx, config, logLevel)
}

func RunPostgresMigrations(ctx context.Context, config config.PostgresConfig, logLevel slog.Level) error {
	dbURL, err := pgutils.GetPostgresURL(config)
	if err != nil {
		return err
	}
	connConfig, err := pgx.ParseConfig(dbURL)
	if err != nil {
		return err
	}
	connConfig.Tracer = pgutils.GetTraceLogger(logLevel)
	db, err := goose.OpenDBWithDriver("pgx", stdlib.RegisterConnConfig(connConfig))
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("goose: failed to close DB", "error", err)
			os.Exit(1)
		}
		slog.Info("goose: closed DB connection for migrations")
	}()
	return goose.UpContext(ctx, db, ".")
}
