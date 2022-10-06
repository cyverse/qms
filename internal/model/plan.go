package model

import (
	"time"
)

// Plan
//
// swagger:model
type Plan struct {

	// The plan identifier
	//
	// readOnly: true
	ID *string `gorm:"type:uuid;default:uuid_generate_v1()" json:"id"`

	// The plan name
	//
	Name string `gorm:"not null;unique" json:"name"`

	// A brief description of the plan
	//
	// required: true
	Description string `gorm:"not null" json:"description"`

	// The default quota values associated with the plan
	PlanQuotaDefaults []PlanQuotaDefault `json:"plan_quota_defaults"`
}

// PlanQuotaDefault define the structure for an Api Plan and Quota.
type PlanQuotaDefault struct {

	// The plan quota default identifier
	//
	// readOnly: true
	ID *string `gorm:"type:uuid;default:uuid_generate_v1()" json:"id"`

	// The plan ID
	PlanID *string `gorm:"type:uuid;not null" json:"-"`

	// The default quota value
	//
	// required: true
	QuotaValue float64 `gorm:"not null" json:"quota_value"`

	// The resource type ID
	ResourceTypeID *string `gorm:"type:uuid;not null" json:"-"`

	// The resource type
	//
	// required: true
	ResourceType ResourceType `json:"resource_type"`
}

// UserPlan define the structure for the API User plans.
type UserPlan struct {
	ID                 *string    `gorm:"type:uuid;default:uuid_generate_v1()" json:"id"`
	EffectiveStartDate *time.Time `gorm:"" json:"effective_start_date"`
	EffectiveEndDate   *time.Time `gorm:"" json:"effective_end_date"`
	UserID             *string    `gorm:"type:uuid;not null" json:"-"`
	User               User       `json:"user"`
	PlanID             *string    `gorm:"type:uuid;not null" json:"-"`
	Plan               Plan       `json:"plan"`
	Quotas             []Quota    `json:"quotas"`
	Usages             []Usage    `json:"usages"`
}

// GetCurrentUsageValue returns the current usage value for the resource type with the given resource type ID. Be
// careful to ensure that all user plan details have been loaded before calling this function.
func (up *UserPlan) GetCurrentUsageValue(resourceTypeID string) float64 {
	var usageValue float64
	for _, usage := range up.Usages {
		if usage.ResourceTypeID == &resourceTypeID {
			usageValue = usage.Usage
		}
	}
	return usageValue
}
