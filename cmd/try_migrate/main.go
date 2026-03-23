package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	pginit "github.com/SwissDataScienceCenter/renku-gateway/internal/pg/init"
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
	slog.Info("loaded config", "config", gwConfig.Postgres)
	err = gwConfig.Postgres.Validate()
	if err != nil {
		slog.Error("the config validation failed", "error", err)
		os.Exit(1)
	}

	err = pginit.RunPostgresMigrations(context.Background(), gwConfig.Postgres, logLevel.Level())
	if err != nil {
		slog.Error("running postgres migrations failed", "error", err)
		os.Exit(1)
	}
}

var logLevel *slog.LevelVar = new(slog.LevelVar)
var jsonLogger *slog.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
