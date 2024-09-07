package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/cyverse-de/go-mod/cfg"
	"github.com/cyverse/qms/config"
	"github.com/cyverse/qms/logging"
	"github.com/cyverse/qms/server"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var log = logging.GetLogger().WithFields(logrus.Fields{"package": "main"})

// runSchemaMigrations runs the schema migrations on the database.
func runSchemaMigrations(dbURI string, reinit bool) error {
	log := log.WithFields(logrus.Fields{"context": "schema migrations"})

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
	var (
		err error

		configPath = flag.String("config", cfg.DefaultConfigPath, "Path to the config file")
		dotEnvPath = flag.String("dotenv-path", cfg.DefaultDotEnvPath, "Path to the dotenv file")
		envPrefix  = flag.String("env-prefix", "QMS_", "The prefix for environment variables")
		logLevel   = flag.String("log-level", "debug", "One of trace, debug, info, warn, error, fatal, or panic.")
	)

	flag.Parse()
	logging.SetupLogging(*logLevel)

	log := log.WithFields(logrus.Fields{"context": "main"})

	// Load the configuration.
	spec, err := config.LoadConfig(*envPrefix, *configPath, *dotEnvPath)
	if err != nil {
		log.Fatalf("unable to load the configuration: %s", err.Error())
	}

	log.Info("loaded the configuration file")

	// Run the schema migrations if we're supposed to.
	if spec.RunSchemaMigrations {
		err = runSchemaMigrations(spec.DatabaseURI, spec.ReinitDB)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	// Initialize the server.
	server.Init(spec)
}
