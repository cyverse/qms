package model

import "time"

// Usage define the structure for API Usages.
type Usage struct {
	ID             *string      `gorm:"type:uuid;default:uuid_generate_v1()" json:"id"`
	Usage          float64      `gorm:"not null" json:"usage"`
	UserPlanID     *string      `gorm:"type:uuid;not null;index:usage_userplan_resourcetype,unique" json:"-"`
	ResourceTypeID *string      `gorm:"type:uuid;not null;index:usage_userplan_resourcetype,unique" json:"-"`
	ResourceType   ResourceType `json:"resource_type"`
	LastModifiedAt *time.Time   `gorm:"->" json:"last_modified_at"`
}
