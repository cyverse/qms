package model

import (
	"time"

	"gorm.io/gorm"
)

type UpdateOperation struct {
	ID   *string `gorm:"type:uuid;default:uuid_generate_v1()" json:"id"`
	Name string  `gorm:"json:name"`
}

func (uo *UpdateOperation) TableName() string {
	return "update_operations"
}

type TrackedMetric struct {
	Quota string `gorm:"json:quota"`
	Usage string `gorm:"json:usage"`
}

type Update struct {
	gorm.Model
	ID                *string      `gorm:"type:uuid;default:uuid_generate_v1()" json:"id"`
	ValueType         string       `gorm:"not null;unique" json:"value_type"`
	Value             float64      `gorm:"not null;unique" json:"value"`
	UpdatedBy         string       `gorm:"not null" json:"updated_by"`
	EffectiveDate     time.Time    `gorm:"json:effective_date;type:date"`
	LastModifiedBy    string       `gorm:"json:last_modified_by"`
	UpdateOperationID *string      `gorm:"type:uuid;not null" json:"-"`
	ResourceTypeID    *string      `gorm:"type:uuid;not null" json:"-"`
	ResourceType      ResourceType `json:"resource_types"`
}
