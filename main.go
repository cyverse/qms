package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cyverse-de/echo-middleware/v2/log"
	"github.com/cyverse-de/go-mod/otelutils"
	"github.com/cyverse/QMS/config"
	"github.com/cyverse/QMS/server"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const serviceName = "QMS"

// buildLoggerEntry sets some logging options then returns a logger entry with some custom fields
// for convenience.
func buildLoggerEntry() *logrus.Entry {

	// Set the logging format to JSON because that's what Echo's middleware uses.
	logrus.SetFormatter(&logrus.JSONFormatter{})

	// Return the custom log entry.
	return logrus.WithFields(logrus.Fields{
		"service": "qms",
		"art-id":  "qms",
		"group":   "org.cyverse",
	})
}

// runSchemaMigrations runs the schema migrations on the database.
func runSchemaMigrations(logger *log.Logger, dbURI string, reinit bool) error {
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
		logger.Info("running the down database migrations")
		err = m.Down()
		if err != nil && err != migrate.ErrNoChange {
			return errors.Wrap(err, wrapMsg)
		}
	}

	// Run the up migrations.
	logger.Info("running the up database migrations")
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return errors.Wrap(err, wrapMsg)
	}

	return nil
}

func main() {
	logger := log.NewLogger(buildLoggerEntry())

	var tracerCtx, cancel = context.WithCancel(context.Background())
	defer cancel()
	shutdown := otelutils.TracerProviderFromEnv(tracerCtx, serviceName, func(e error) { logger.Fatal(e) })
	defer shutdown()

	// Load the configuration.
	spec, err := config.LoadConfig()
	if err != nil {
		logger.Fatalf("unable to load the configuration: %s", err.Error())
	}

	// Run the schema migrations.
	err = runSchemaMigrations(logger, spec.DatabaseURI, spec.ReinitDB)
	if err != nil {
		logger.Fatal(err.Error())
	}

	// Initialize the server.
	server.Init(logger, spec)
}
