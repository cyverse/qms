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

func natsSubject(base string, fields ...string) string {
	trimmed := strings.TrimSuffix(
		strings.TrimSuffix(base, ".*"),
		".>",
	)
	addFields := strings.Join(fields, ".")
	return fmt.Sprintf("%s.%s", trimmed, addFields)
}

func natsQueue(qBase string, fields ...string) string {
	return fmt.Sprintf("%s.%s", qBase, strings.Join(fields, "."))
}

func queueSub(conn *nats.EncodedConn, spec *config.Specification, name string, handler nats.Handler) {
	var err error

	subject := natsSubject(spec.BaseSubject, name)
	queue := natsQueue(spec.BaseQueueName, name)

	if _, err = conn.QueueSubscribe(subject, queue, handler); err != nil {
		log.Fatal(err)
	}

	log.Infof("subscribed to %s on queue %s", subject, queue)
}

func InitNATS(spec *config.Specification) *nats.EncodedConn {
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

	return conn
}

func Init(spec *config.Specification) {
	log := log.WithFields(logrus.Fields{"context": "server init"})

	e := InitRouter()

	// Establish the database connection.
	log.Info("establishing the database connection")
	db, gormdb, err := db.Init("postgres", spec.DatabaseURI)
	if err != nil {
		log.Fatalf("service initialization failed: %s", err.Error())
	}

	conn := InitNATS(spec)

	s := controllers.Server{
		Router:         e,
		DB:             db,
		GORMDB:         gormdb,
		Service:        "qms",
		Title:          "serviceInfo.Title",   //TODO: correct this
		Version:        "serviceInfo.Version", //TODO:correct this
		NATSConn:       conn,
		ReportOverages: spec.ReportOverages,
		UsernameSuffix: spec.UsernameSuffix,
	}

	// Register the handlers.
	RegisterHandlers(s)

	queueSub(conn, spec, "user.overages.get", s.GetUserOveragesNATS)
	queueSub(conn, spec, "user.overages.check", s.InOverageNATS)

	// Only use this endpoint if you need to modify a usage directly, bypassing
	// the updates table.
	queueSub(conn, spec, "user.usages.add", s.AddUsagesNATS)

	// This should be safe to use to get the current usages.
	queueSub(conn, spec, "user.usages.get", s.GetUsagesNATS)

	queueSub(conn, spec, "user.updates.get", s.GetAllUpdatesForUser)
	queueSub(conn, spec, "user.updates.add", s.AddUpdateForUser)

	log.Info("starting the service")
	log.Fatal(e.Start(fmt.Sprintf(":%d", 9000)))
}
