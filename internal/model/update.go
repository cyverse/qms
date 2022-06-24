package model

import (
	"time"
)

// UpdateOperation defines the structure of an available update operation in the QMS database.
//
// swagger:model
type UpdateOperation struct {
	// The update operation ID
	//
	// readOnly: true
	ID *string `gorm:"type:uuid;default:uuid_generate_v1()" json:"id"`
	// The update operation name
	//
	// required: true
	Name string `gorm:"type:text;not null;unique" json:"name"`
}

type Update struct {
	ID                *string      `gorm:"type:uuid;default:uuid_generate_v1()" json:"id"`
	ValueType         string       `json:"value_type"`
	Value             float64      `gorm:"not null" json:"value"`
	EffectiveDate     time.Time    `gorm:"type:date;not null" json:"effective_date"`
	UpdateOperationID *string      `gorm:"type:uuid;not null" json:"-"`
	ResourceTypeID    *string      `gorm:"type:uuid;not null" json:"-"`
	ResourceType      ResourceType `json:"resource_types"`
}
