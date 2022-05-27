package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/cyverse/QMS/config"
	"github.com/cyverse/QMS/internal/controllers"
	"github.com/cyverse/QMS/internal/db"
	"github.com/cyverse/QMS/logging"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

var log = logging.GetLogger().WithFields(logrus.Fields{"package": "server"})

func Init(spec *config.Specification) {
	log := log.WithFields(logrus.Fields{"context": "server init"})

	e := InitRouter()

	// Establish the database connection.
	log.Info("establishing the database connection")
	db, gormdb, err := db.Init("postgres", spec.DatabaseURI)
	if err != nil {
		log.Fatalf("service initialization failed: %s", err.Error())
	}

	s := controllers.Server{
		Router:  e,
		DB:      db,
		GORMDB:  gormdb,
		Service: "qms",
		Title:   "serviceInfo.Title",   //TODO: correct this
		Version: "serviceInfo.Version", //TODO:correct this
	}

	// Register the handlers.
	RegisterHandlers(s)
	log.Info("starting the service")
	log.Fatal(e.Start(fmt.Sprintf(":%d", 9000)))
}

func InitNATS(spec *config.Specification) {
	nc, err := nats.Connect(
		spec.NatsCluster,
		nats.UserCredentials(spec.CredsPath),
		nats.RootCAs(spec.CACertPath),
		nats.ClientCert(spec.TLSCertPath, spec.TLSKeyPath),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(spec.MaxReconnects),
		nats.ReconnectWait(time.Duration(spec.ReconnectWait)*time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Errorf("disconnected from nats: %s", err.Error())
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Infof("reconnected to %s", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Errorf("connection closed: %s", nc.LastError().Error())
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("configured servers: %s", strings.Join(nc.Servers(), " "))
	log.Infof("connected to NATS host: %s", nc.ConnectedServerName())

	conn, err := nats.NewEncodedConn(nc, "protojson")
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("set up encoded connection to NATS")

	// TODO: add logic for routing based on subject. Generate queue names.
	if _, err = conn.QueueSubscribe(spec.BaseSubject, spec.BaseQueueName, GetNATSHandler()); err != nil {
		log.Fatal(err)
	}

	log.Infof("subscribed to NATS queue")
}

func GetNATSHandler() nats.Handler {
	return nil
}
