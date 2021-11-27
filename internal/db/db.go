package db

import (
	"database/sql"

	"github.com/cyverse-de/dbutil"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// InitDatabase establishes a database connection and verifies that the database can be reached.
func Init(driverName, databaseURI string) (*sql.DB, *gorm.DB, error) {

	wrapMsg := "unable to initialize the database"

	// Create a database connector to establish the connection for us.
	connector, err := dbutil.NewDefaultConnector("1m")
	if err != nil {
		return nil, nil, errors.Wrap(err, wrapMsg)
	}

	// Establish the database connection.
	db, err := connector.Connect(driverName, databaseURI)
	if err != nil {
		return nil, nil, errors.Wrap(err, wrapMsg)
	}
	gormdb, err := InitGORMConnection(db)
	if err != nil {
		return nil, nil, errors.Wrap(err, wrapMsg)
	}
	return db, gormdb, nil
}
