package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cyverse-de/go-mod/otelutils"
	"github.com/cyverse/QMS/config"
	"github.com/cyverse/QMS/logging"
	"github.com/cyverse/QMS/server"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/pkg/errors"
)

const serviceName = "QMS"

var log = logging.Log

func init() {
	logging.SetServiceName(serviceName)
}

// runSchemaMigrations runs the schema migrations on the database.
func runSchemaMigrations(dbURI string, reinit bool) error {
	wrapMsg := "unable to run the schema migrations"

	// Build the URI to the migrations.
	workingDir, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, wrapMsg)
	}
	migrationsURI := fmt.Sprintf("file://%s/migrations", workingDir)

	// Initialize the migrations.
	m, err := migrate.New(migrationsURI, dbURI)
	if err != nil {
		return errors.Wrap(err, wrapMsg)
	}

	// Run the down migrations if we're supposed to.
	if reinit {
		log.Info("running the down database migrations")
		err = m.Down()
		if err != nil && err != migrate.ErrNoChange {
			return errors.Wrap(err, wrapMsg)
		}
	}

	// Run the up migrations.
	log.Info("running the up database migrations")
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return errors.Wrap(err, wrapMsg)
	}

	return nil
}

func main() {
	var tracerCtx, cancel = context.WithCancel(context.Background())
	defer cancel()
	shutdown := otelutils.TracerProviderFromEnv(tracerCtx, serviceName, func(e error) { log.Fatal(e) })
	defer shutdown()

	// Load the configuration.
	spec, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("unable to load the configuration: %s", err.Error())
	}

	// Run the schema migrations.
	err = runSchemaMigrations(spec.DatabaseURI, spec.ReinitDB)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Initialize the server.
	server.Init(spec)
}
