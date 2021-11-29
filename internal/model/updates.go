package model

import (
	"time"

	"gorm.io/gorm"
)

type UpdateOperations struct {
	ID   string `gorm:"column:id;primaryKey;type:uuid"`
	Name string `gorm:"column:name;type:varchar(100)"`
}

func (uo *UpdateOperations) TableName() string {
	return "update_operations"
}

type TrackedMetrics struct {
	Quota string `gorm:"column:quota;type:varchar(100)"`
	Usage string `gorm:"column:usage;type:varchar(100)"`
}

func (tm *TrackedMetrics) TableName() string {
	return "tracked_metrics"
}

type Updates struct {
	gorm.Model
	ID               string    `gorm:"column:id;primaryKey;type:uuid"`
	ValueType        string    `gorm:"column:value_type;type:varchar(100)"`
	Value            float64   `gorm:"coumn:value;type:numeric"`
	UpdatedBy        string    `gorm:"column:updated_by;type:varchar(100)"`
	EffectiveDate    time.Time `gorm:"column:effective_date;type:date"`
	LastModifiedBy   string    `gorm:"column:last_modified_by;varchar(100)"`
	OpID             string
	UpdateOperations UpdateOperations `gorm:"foreignKey:OpID;references:ID;"`
	ResourceTypeID   string
	ResourceTypes    ResourceTypes `gorm:"foreignKey:ResourceTypeID;references:ID"`
}

func (u *Updates) TableName() string {
	return "updates"
}
