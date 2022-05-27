package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/cyverse-de/go-mod/cfg"
	"github.com/cyverse-de/go-mod/gotelnats"
	"github.com/cyverse-de/go-mod/otelutils"
	"github.com/cyverse/QMS/config"
	"github.com/cyverse/QMS/logging"
	"github.com/cyverse/QMS/server"
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

		configPath    = flag.String("config", cfg.DefaultConfigPath, "Path to the config file")
		dotEnvPath    = flag.String("dotenv-path", cfg.DefaultDotEnvPath, "Path to the dotenv file")
		tlsCert       = flag.String("tlscert", gotelnats.DefaultTLSCertPath, "Path to the NATS TLS cert file")
		tlsKey        = flag.String("tlskey", gotelnats.DefaultTLSKeyPath, "Path to the NATS TLS key file")
		caCert        = flag.String("tlsca", gotelnats.DefaultTLSCAPath, "Path to the NATS TLS CA file")
		credsPath     = flag.String("creds", gotelnats.DefaultCredsPath, "Path to the NATS creds file")
		maxReconnects = flag.Int("max-reconnects", gotelnats.DefaultMaxReconnects, "Maximum number of reconnection attempts to NATS")
		reconnectWait = flag.Int("reconnect-wait", gotelnats.DefaultReconnectWait, "Seconds to wait between reconnection attempts to NATS")
		natsSubject   = flag.String("subject", "discoenv.qms.>", "NATS subject to subscribe to")
		natsQueue     = flag.String("queue", "discoenv_qms", "Name of the NATS queue to use")
		envPrefix     = flag.String("env-prefix", "QMS_", "The prefix for environment variables")
	)

	log := log.WithFields(logrus.Fields{"context": "main"})

	var tracerCtx, cancel = context.WithCancel(context.Background())
	defer cancel()
	shutdown := otelutils.TracerProviderFromEnv(tracerCtx, config.ServiceName, func(e error) { log.Fatal(e) })
	defer shutdown()

	// Load the configuration.
	spec, err := config.LoadConfig(*envPrefix, *configPath, *dotEnvPath)
	if err != nil {
		log.Fatalf("unable to load the configuration: %s", err.Error())
	}

	spec.CACertPath = *caCert
	spec.TLSCertPath = *tlsCert
	spec.TLSKeyPath = *tlsKey
	spec.CredsPath = *credsPath
	spec.BaseQueueName = *natsQueue
	spec.BaseSubject = *natsSubject
	spec.MaxReconnects = *maxReconnects
	spec.ReconnectWait = *reconnectWait

	log.Infof("NATS URLs are %s", spec.NatsCluster)
	log.Infof("NATS TLS cert file is %s", *tlsCert)
	log.Infof("NATS TLS key file is %s", *tlsKey)
	log.Infof("NATS CA cert file is %s", *caCert)
	log.Infof("NATS creds file is %s", *credsPath)
	log.Infof("NATS subject is %s", *natsSubject)
	log.Infof("NATS queue is %s", *natsQueue)

	// Run the schema migrations.
	err = runSchemaMigrations(spec.DatabaseURI, spec.ReinitDB)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Initialize the NATS connection. Does not block.
	server.InitNATS(spec)

	// Initialize the server.
	server.Init(spec)
}
