package server

import (
	"fmt"

	"github.com/cyverse/qms/config"
	"github.com/cyverse/qms/internal/controllers"
	"github.com/cyverse/qms/internal/db"
	"github.com/cyverse/qms/logging"
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
		Router:         e,
		DB:             db,
		GORMDB:         gormdb,
		Service:        "qms",
		Title:          "serviceInfo.Title",   //TODO: correct this
		Version:        "serviceInfo.Version", //TODO:correct this
		UsernameSuffix: spec.UsernameSuffix,
	}

	// Register the handlers.
	RegisterHandlers(s)

	log.Info("starting the service")
	log.Fatal(e.Start(fmt.Sprintf(":%d", 9000)))
}
