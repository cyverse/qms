package httpmodel

import (
	"fmt"

	"github.com/cyverse/qms/internal/model"
)

// Note: the names in the comments may deviate a bit from the actual structure names in order to avoid producing
// confusing Swagger docs.

// NewPlan
//
// swagger:model
type NewPlan struct {

	// The plan name
	//
	// required: true
	Name string `json:"name"`

	// A brief description of the plan
	//
	// required: true
	Description string `json:"description"`

	// The default quota values associated with the plan
	PlanQuotaDefaults []NewPlanQuotaDefault `json:"plan_quota_defaults"`
}

// Validate verifies that all the required fields in a new plan are present.
func (p NewPlan) Validate() error {
	var err error

	// The plan name and description are both required.
	if p.Name == "" {
		return fmt.Errorf("a plan name is required")
	}
	if p.Description == "" {
		return fmt.Errorf("a plan description is required")
	}

	// Validate each of the default quota values.
	for _, d := range p.PlanQuotaDefaults {
		err = d.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

// ToDBModel converts a plan to its equivalent database model.
func (p NewPlan) ToDBModel() model.Plan {

	// Convert each of the plan quota defaults.
	planQuotaDefaults := make([]model.PlanQuotaDefault, len(p.PlanQuotaDefaults))
	for i, planQuotaDefault := range p.PlanQuotaDefaults {
		planQuotaDefaults[i] = planQuotaDefault.ToDBModel()
	}

	return model.Plan{
		Name:              p.Name,
		Description:       p.Description,
		PlanQuotaDefaults: planQuotaDefaults,
	}
}

// NewPlanQuotaDefault
//
// swagger:model
type NewPlanQuotaDefault struct {

	// The plan ID
	PlanID *string `json:"-"`

	// The default quota value
	//
	// required: true
	QuotaValue float64 `json:"quota_value"`

	// The resource type ID
	ResourceTypeID *string `json:"-"`

	// The resource type
	//
	// required: true
	ResourceType NewPlanResourceType `json:"resource_type"`
}

// Validate verifies that all the required fields in a quota default are present.
func (d NewPlanQuotaDefault) Validate() error {

	// The default quota value is required.
	if d.QuotaValue <= 0 {
		return fmt.Errorf("default quota values must be specified and greater than zero")
	}

	return d.ResourceType.Validate()
}

// ToDBModel converts a plan quota default to its equivalent database model.
func (d NewPlanQuotaDefault) ToDBModel() model.PlanQuotaDefault {
	return model.PlanQuotaDefault{
		QuotaValue:   d.QuotaValue,
		ResourceType: d.ResourceType.ToDBModel(),
	}
}

// NewPlanResourceType
//
// swagger:model
type NewPlanResourceType struct {

	// The resource type name
	//
	// required: true
	Name string `json:"name"`
}

// Validate verifies that all the required fields in a resource type are present.
func (rt NewPlanResourceType) Validate() error {

	// The resource type name is required.
	if rt.Name == "" {
		return fmt.Errorf("the resource type name is required")
	}

	return nil
}

// ToDBModel converts a resource type to its equivalent database model.
func (rt NewPlanResourceType) ToDBModel() model.ResourceType {
	return model.ResourceType{Name: rt.Name}
}
