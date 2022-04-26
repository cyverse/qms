package db

import (
	"database/sql"
	"errors"

	"github.com/uptrace/opentelemetry-go-extra/otelgorm"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitGORMConnection(db *sql.DB) (*gorm.DB, error) {
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}))
	if err != nil {
		return gormDB, errors.New("failed to connect database")
	}

	err = gormDB.Use(otelgorm.NewPlugin())
	if err != nil {
		return gormDB, errors.New("failed to set up opentelemetry")
	}

	return gormDB, nil
}
