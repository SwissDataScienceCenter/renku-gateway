package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"

	_ "github.com/SwissDataScienceCenter/renku-gateway/internal/pg/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

var (
	flags = flag.NewFlagSet("goose", flag.ExitOnError)
	dir   = flags.String("dir", "internal/pg/migrations", "directory with migration files")
)

func main() {
	if err := flags.Parse(os.Args[1:]); err != nil {
		log.Fatalf("goose: failed to parse flags: %v", err)
	}
	args := flags.Args()

	if len(args) < 1 {
		flags.Usage()
		os.Exit(1)
	}

	dbURL, err := getPostgresURL()
	if err != nil {
		log.Fatalf("goose: could not form postgres database URL: %v\n", err)
	}

	db, err := goose.OpenDBWithDriver("pgx", dbURL)
	if err != nil {
		log.Fatalf("goose: unable to connect to database: %v", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Fatalf("goose: failed to close DB: %v", err)
		}
	}()

	command := args[0]
	arguments := []string{}
	if len(args) > 1 {
		arguments = append(arguments, args[1:]...)
	}

	ctx := context.Background()
	if err := goose.RunContext(ctx, command, db, *dir, arguments...); err != nil {
		log.Fatalf("goose %v: %v", command, err)
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
