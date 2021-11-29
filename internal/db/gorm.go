package db

import (
	"database/sql"
	"errors"

	"github.com/cyverse/QMS/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitGORMConnection(db *sql.DB) (*gorm.DB, error) {
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return gormDB, errors.New("failed to connect database")
	}
	return gormDB, nil
}

func MigrateTables(db *gorm.DB) error {
	err := db.AutoMigrate(&model.Users{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&model.ResourceTypes{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&model.Plans{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&model.PlanQuotaDefaults{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&model.UserPlans{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&model.Quotas{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&model.UpdateOperations{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&model.TrackedMetrics{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&model.Updates{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&model.Usages{})
	if err != nil {
		return err
	}
	return nil
}
