package db

import (
	"database/sql"

	"github.com/cyverse-de/dbutil"
	_ "github.com/lib/pq"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// InitDatabase establishes a database connection and verifies that the database can be reached.
func Init(driverName, databaseURI string) (*sql.DB, *gorm.DB, error) {

	wrapMsg := "unable to initialize the database"

	/*conn, err := sql.Open(driverName, databaseURI)
	if err != nil {
		return nil, nil, errors.Wrap(err, wrapMsg)
	}

	if err := conn.Ping(); err != nil {
		return nil, nil, errors.Wrap(err, wrapMsg)
	}
	conn.SetMaxIdleConns(5)
	conn.SetMaxOpenConns(30)
	conn.SetConnMaxLifetime(time.Hour)
	*/

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

	err = MigrateTables(gormdb)
	if err != nil {
		return nil, nil, errors.Wrap(err, wrapMsg)
	}
	return conn, gormdb, nil

}
