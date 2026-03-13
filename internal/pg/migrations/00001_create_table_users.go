package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upExample, downExample)
}

func upExample(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.ExecContext(ctx, "CREATE SCHEMA IF NOT EXISTS gateway")
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS gateway.users (
		id TEXT PRIMARY KEY NOT NULL,
		refresh_token TEXT DEFAULT NULL,
		refresh_expires_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
		last_activity TIMESTAMP WITH TIME ZONE DEFAULT NULL
	)`)
	return err
}

func downExample(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.ExecContext(ctx, "DROP TABLE IF EXISTS gateway.users")
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "DROP SCHEMA IF EXISTS gateway")
	return err
}
