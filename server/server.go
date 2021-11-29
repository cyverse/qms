package server

import (
	"fmt"

	"github.com/cyverse/QMS/internal/controllers"
	"github.com/cyverse/QMS/internal/db"
	"github.com/labstack/gommon/log"
)

func Init() {

	e := InitRouter()

	// Establish the database connection.
	log.Info("establishing the database connection")

	databaseURI := "postgres://postgres:password@localhost:54320/qmsdb?sslmode=disable" //cfg.GetString("notifications.db.uri")

	db, gormdb, err := db.Init("postgres", databaseURI)
	if err != nil {
		e.Logger.Fatalf("service initialization failed: %s", err.Error())
	}

	// Define the primary API handler.
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

	// Start the service.
	e.Logger.Info("starting the service")
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", 9000))) //TODO: get the value form config
}
