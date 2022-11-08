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
	ID *string `gorm:"type:uuid;default:uuid_generate_v1()" json:"id,omitempty"`

	// The plan name
	Name string `gorm:"not null;unique" json:"name,omitempty"`

	// A brief description of the plan
	//
	// required: true
	Description string `gorm:"not null" json:"description,omitempty"`

	// The default quota values associated with the plan
	PlanQuotaDefaults []PlanQuotaDefault `json:"plan_quota_defaults,omitempty"`
}

// GetDefaultQuotaValue returns the default quota value associated with the resource type with the given name.
func (p *Plan) GetDefaultQuotaValue(resourcetypeName string) float64 {
	var value float64
	for _, quotaDefault := range p.PlanQuotaDefaults {
		if quotaDefault.ResourceType.Name == resourcetypeName {
			value = quotaDefault.QuotaValue
			break
		}
	}
	return value
}

// PlanQuotaDefault define the structure for an Api Plan and Quota.
type PlanQuotaDefault struct {
	// The plan quota default identifier
	//
	// readOnly: true
	ID *string `gorm:"type:uuid;default:uuid_generate_v1()" json:"id,omitempty"`

	// The plan ID
	PlanID *string `gorm:"type:uuid;not null" json:"-"`

	// The default quota value
	//
	// required: true
	QuotaValue float64 `gorm:"not null" json:"quota_value,omitempty"`

	// The resource type ID
	ResourceTypeID *string `gorm:"type:uuid;not null" json:"-"`

	// The resource type
	//
	// required: true
	ResourceType ResourceType `json:"resource_type,omitempty"`
}

// UserPlan define the structure for the API User plans.
//
// swagger:model
type UserPlan struct {
	// The subscription identifier
	//
	// readOnly: true
	ID *string `gorm:"type:uuid;default:uuid_generate_v1()" json:"id,omitempty"`

	// The date and time the subscription becomes active
	EffectiveStartDate *time.Time `gorm:"" json:"effective_start_date,omitempty"`

	// The date and time the subscription expires
	EffectiveEndDate *time.Time `gorm:"" json:"effective_end_date,omitempty"`

	// The user identifier
	UserID *string `gorm:"type:uuid;not null" json:"-"`

	// The user associated with the subscription
	User *User `json:"user,omitempty"`

	// The identifier of the plan associated with the subscription
	PlanID *string `gorm:"type:uuid;not null" json:"-"`

	// The plan associated with the subscription
	Plan *Plan `json:"plan,omitempty"`

	// The quotas associated with the subscription
	Quotas []Quota `json:"quotas,omitempty"`

	// The recorded usage amounts associated witht he subscription
	Usages []Usage `json:"usages,omitempty"`
}

// GetCurrentUsageValue returns the current usage value for the resource type with the given resource type ID. Be
// careful to ensure that all user plan details have been loaded before calling this function.
func (up *UserPlan) GetCurrentUsageValue(resourceTypeID string) float64 {
	var usageValue float64
	for _, usage := range up.Usages {
		if *usage.ResourceTypeID == resourceTypeID {
			usageValue = usage.Usage
		}
	}
	return usageValue
}
