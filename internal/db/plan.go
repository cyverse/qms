package db

import (
	"context"
	"fmt"

	"github.com/cyverse/qms/internal/model"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const (
	PlanNameBasic = "Basic"
)

// GetPlan looks up the plan with the given name.
func GetPlan(ctx context.Context, db *gorm.DB, planName string) (*model.Plan, error) {
	wrapMsg := fmt.Sprintf("unable to look up plan name '%s'", planName)
	var err error
	var plan = model.Plan{}
	err = db.
		WithContext(ctx).
		Where("name=?", planName).
		Preload("PlanQuotaDefaults").
		Preload("PlanQuotaDefaults.ResourceType").
		First(&plan).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, wrapMsg)
	}
	return &plan, nil
}

// GetPlanByID looks up the plan with the given identifier.
func GetPlanByID(ctx context.Context, db *gorm.DB, planID string) (*model.Plan, error) {
	wrapMsg := fmt.Sprintf("unable to look up plan ID `%s'", planID)
	var err error

	plan := model.Plan{ID: &planID}
	err = db.
		WithContext(ctx).
		Preload("PlanQuotaDefaults").
		Preload("PlanQuotaDefaults.ResourceType").
		First(&plan).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, wrapMsg)
	}

	return &plan, nil
}

// ListPlans lists all of the plans that are currently available.
func ListPlans(ctx context.Context, db *gorm.DB) ([]*model.Plan, error) {
	wrapMsg := "unable to list plans"
	var err error

	// List the plans.
	var plans []*model.Plan
	err = db.
		WithContext(ctx).
		Preload("PlanQuotaDefaults").
		Preload("PlanQuotaDefaults.ResourceType").
		Find(&plans).Error
	if err != nil {
		return nil, errors.Wrapf(err, wrapMsg)
	}

	return plans, nil
}

func GetDefaultQuotaForPlan(ctx context.Context, db *gorm.DB, planID string) ([]model.PlanQuotaDefault, error) {
	wrapMsg := "unable to look up plan name "
	var err error

	var planQuotaDefaults []model.PlanQuotaDefault
	err = db.WithContext(ctx).Model(&planQuotaDefaults).Where("plan_id=?", planID).Scan(&planQuotaDefaults).Error
	//err = db.Find(&planQuotaDefaults).Error
	if err != nil {
		return nil, errors.Wrap(err, wrapMsg)
	}

	return planQuotaDefaults, nil
}

// GetPlansByName builds a map from plan name to plan details.
func GetPlansByName(ctx context.Context, db *gorm.DB) (map[string]*model.Plan, error) {
	plans, err := ListPlans(ctx, db)
	if err != nil {
		return nil, err
	}

	// Build the map from the plan name to the plan details.
	result := make(map[string]*model.Plan)
	for _, plan := range plans {
		result[plan.Name] = plan
	}

	return result, nil
}
