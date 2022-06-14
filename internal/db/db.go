package db

import (
	"database/sql"

	"github.com/cyverse-de/dbutil"
	_ "github.com/lib/pq" // imported for postgres database suport

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// Init establishes a database connection and verifies that the database can be reached.
func Init(driverName, databaseURI string) (*sql.DB, *gorm.DB, error) {

	wrapMsg := "unable to initialize the database"
	// Create a database connector to establish the connection for us.
	connector, err := dbutil.NewDefaultConnector("5s")
	if err != nil {
		return nil, nil, errors.Wrap(err, wrapMsg)
	}
	// Establish the database connection.
	conn, err := connector.Connect(driverName, databaseURI)
	if err != nil {
		return nil, nil, errors.Wrap(err, wrapMsg)
	}
	gormdb, err := InitGORMConnection(conn)
	if err != nil {
		return nil, nil, errors.Wrap(err, wrapMsg)
	}

	return conn, gormdb, nil
}
