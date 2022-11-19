package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cyverse/QMS/internal/model"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// QuotasFromPlan generates a set of quotas from the plan quota defaults in a plan. This function assumes that the
// given plan already contains the plan quota defaults.
func QuotasFromPlan(plan *model.Plan) []model.Quota {
	result := make([]model.Quota, len(plan.PlanQuotaDefaults))
	for i, quotaDefault := range plan.PlanQuotaDefaults {
		result[i] = model.Quota{
			Quota:          quotaDefault.QuotaValue,
			ResourceTypeID: quotaDefault.ResourceTypeID,
		}
	}
	return result
}

// SubscribeUserToPlan subscribes the given user to the given plan.
func SubscribeUserToPlan(ctx context.Context, db *gorm.DB, user *model.User, plan *model.Plan) (*model.UserPlan, error) {
	wrapMsg := "unable to add user plan"
	var err error

	// Define the user plan.
	effectiveStartDate := time.Now()
	effectiveEndDate := effectiveStartDate.AddDate(1, 0, 0)
	userPlan := model.UserPlan{
		EffectiveStartDate: &effectiveStartDate,
		EffectiveEndDate:   &effectiveEndDate,
		UserID:             user.ID,
		PlanID:             plan.ID,
		Quotas:             QuotasFromPlan(plan),
	}
	err = db.WithContext(ctx).Create(&userPlan).Error
	if err != nil {
		return nil, errors.Wrap(err, wrapMsg)
	}

	return &userPlan, nil
}

// SubscribeUserToDefaultPlan adds the default user plan to the given user.
func SubscribeUserToDefaultPlan(ctx context.Context, db *gorm.DB, username string) (*model.UserPlan, error) {
	wrapMsg := "unable to add the default user plan"
	var err error

	// Get the user ID.
	user, err := GetUser(ctx, db, username)
	if err != nil {
		return nil, errors.Wrap(err, wrapMsg)
	}

	// Get the basic plan ID.
	plan, err := GetPlan(ctx, db, PlanNameBasic)
	if err != nil {
		return nil, errors.Wrap(err, wrapMsg)
	}

	// Subscribe the user to the plan.
	return SubscribeUserToPlan(ctx, db, user, plan)
}

// GetActiveUserPlan retrieves the user plan record that is currently active for the user. The effective start
// date must be before the current date and the effective end date must either be null or after the current date.
// If multiple active user plans exist, the one with the most recent effective start date is used. If no active
// user plans exist for the user then a new one for the basic plan is created.
func GetActiveUserPlan(ctx context.Context, db *gorm.DB, username string) (*model.UserPlan, error) {
	wrapMsg := "unable to get the active user plan"
	var err error

	// Look up the currently active user plan, adding a new one if it doesn't exist already.
	var userPlan model.UserPlan
	err = db.
		WithContext(ctx).
		Table("user_plans").
		Joins("JOIN users ON user_plans.user_id=users.id").
		Where("users.username=?", username).
		Where(
			db.Where("CURRENT_TIMESTAMP BETWEEN user_plans.effective_start_date AND user_plans.effective_end_date").
				Or("CURRENT_TIMESTAMP > user_plans.effective_start_date AND user_plans.effective_end_date IS NULL"),
		).
		Order("user_plans.effective_start_date desc").
		First(&userPlan).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.Wrap(err, wrapMsg)
	} else if err == gorm.ErrRecordNotFound {
		userPlanPtr, err := SubscribeUserToDefaultPlan(ctx, db, username)
		if err != nil {
			return nil, errors.Wrap(err, wrapMsg)
		}
		userPlan = *userPlanPtr
	}

	return &userPlan, nil
}

// GetUserPlanDetails loads the details for the user plan with the given ID from the database. This function assumes
// that the user plan exists.
func GetUserPlanDetails(ctx context.Context, db *gorm.DB, userPlanID string) (*model.UserPlan, error) {
	var userPlan *model.UserPlan

	err := db.WithContext(ctx).
		Preload("User").
		Preload("Plan").
		Preload("Plan.PlanQuotaDefaults").
		Preload("Plan.PlanQuotaDefaults.ResourceType").
		Preload("Quotas").
		Preload("Quotas.ResourceType").
		Preload("Usages").
		Preload("Usages.ResourceType").
		Where("id = ?", userPlanID).
		First(&userPlan).
		Error

	return userPlan, err
}

// UserPlanListingParams represents the parameters that can be used to customize a user plan listing.
type UserPlanListingParams struct {
	Offset    int
	Limit     int
	SortField string
	SortOrder string
	Search    string
}

// ListUserPlans lists subscriptions for multiple users.
func ListUserPlans(ctx context.Context, db *gorm.DB, params *UserPlanListingParams) ([]*model.UserPlan, int64, error) {
	var userPlans []*model.UserPlan
	var count int64

	// Determine the offset and limit to use.
	var offset int = 0
	if params != nil && params.Offset >= 0 {
		offset = params.Offset
	}
	var limit int = 50
	if params != nil && params.Limit >= 0 {
		limit = params.Limit
	}

	// Determine the sort field and sort order to use.
	sortField := "users.username"
	if params != nil && params.SortField != "" {
		sortField = params.SortField
	}
	order := "asc"
	if params != nil && params.SortOrder != "" {
		order = params.SortOrder
	}
	orderBy := fmt.Sprintf("%s %s", sortField, order)

	// Build the base query.
	baseQuery := db.WithContext(ctx).
		Joins("JOIN users ON user_plans.user_id=users.id").
		Preload("User").
		Preload("Plan").
		Preload("Plan.PlanQuotaDefaults").
		Preload("Plan.PlanQuotaDefaults.ResourceType").
		Preload("Quotas").
		Preload("Quotas.ResourceType").
		Preload("Usages").
		Preload("Usages.ResourceType").
		Where(
			db.Where("CURRENT_TIMESTAMP BETWEEN user_plans.effective_start_date AND user_plans.effective_end_date").
				Or("CURRENT_TIMESTAMP > user_plans.effective_start_date AND user_plans.effective_end_date IS NULL"),
		)

	// Add the search clause if we're supposed to.
	if params.Search != "" {
		search := strings.ReplaceAll(params.Search, "%", "\\%")
		search = strings.ReplaceAll(search, "_", "\\_")
		baseQuery = baseQuery.Where("users.username LIKE ?", "%"+search+"%")
	}

	// Count the number of items in the result set.
	err := baseQuery.
		Model(&userPlans).
		Count(&count).Error

	// Look up the result set.
	if err == nil {
		err = baseQuery.
			Offset(offset).
			Limit(limit).
			Order(orderBy).
			Find(&userPlans).Error
	}

	return userPlans, count, err
}

// GetActiveUserPlanDetails retrieves the user plan information that is currently active for the user. The effective
// start date must be before the current date and the effective end date must either be null or after the current date.
// If multiple active user plans exist, the one with the most recent effective start date is used. If no active user
// plans exist for the user then a new one for the basic plan is created. This funciton is like GetActiveUserPlan except
// that it also loads all of the user plan details from the database.
func GetActiveUserPlanDetails(ctx context.Context, db *gorm.DB, username string) (*model.UserPlan, error) {
	var err error

	// Get the current user plan.
	userPlan, err := GetActiveUserPlan(ctx, db, username)
	if err != nil {
		return nil, err
	}

	// Load the user plan details.
	return GetUserPlanDetails(ctx, db, *userPlan.ID)
}

// DeactivateUserPlans marks all currently active plans for a user as expired. This operation is used when a user
// selects a new plan. This function does not support user plans that become active in the future at this time.
func DeactivateUserPlans(ctx context.Context, db *gorm.DB, userID string) error {
	wrapMsg := "unable to deactivate active plans for user"
	// Mark currently active user plans as expired.
	err := db.WithContext(ctx).
		Model(&model.UserPlan{}).
		Select("EffectiveEndDate").
		Where("user_id = ?", userID).
		Where("effective_end_date > CURRENT_TIMESTAMP").
		UpdateColumn("effective_end_date", gorm.Expr("CURRENT_TIMESTAMP")).
		Error
	if err != nil {
		return errors.Wrap(err, wrapMsg)
	}
	return nil
}

func GetUserOverages(ctx context.Context, db *gorm.DB, username string) ([]map[string]interface{}, error) {
	var err error

	retval := make([]map[string]interface{}, 0)

	err = db.WithContext(ctx).
		Table("user_plans").
		Select(
			"user_plans.id as user_plan_id",
			"users.username",
			"plans.name as plan_name",
			"resource_types.name as resource_type_name",
			"quotas.quota",
			"usages.usage",
		).
		Joins("JOIN users ON user_plans.user_id = users.id").
		Joins("JOIN plans ON user_plans.plan_id = plans.id").
		Joins("JOIN quotas ON user_plans.id = quotas.user_plan_id").
		Joins("JOIN usages ON user_plans.id = usages.user_plan_id").
		Joins("JOIN resource_types ON usages.resource_type_id = resource_types.id").
		Where("users.username = ?", username).
		Where(
			db.Where("CURRENT_TIMESTAMP BETWEEN user_plans.effective_start_date AND user_plans.effective_end_date").
				Or("CURRENT_TIMESTAMP > user_plans.effective_start_date AND user_plans.effective_end_date IS NULL"),
		).
		Where("usages.resource_type_id = quotas.resource_type_id").
		Where("usages.usage >= quotas.quota").
		Find(&retval).Error

	if err != nil {
		return nil, errors.Wrap(err, "failed to look up overages")
	}

	return retval, nil
}

func IsOverage(ctx context.Context, db *gorm.DB, username string, resourceName string) (map[string]interface{}, error) {
	var err error

	rsc, err := GetResourceTypeByName(ctx, db, resourceName)
	if err != nil {
		return nil, err
	}
	if rsc == nil {
		return nil, fmt.Errorf("resource type %s does not exist", resourceName)
	}

	result := make([]map[string]interface{}, 0)
	retval := make(map[string]interface{})

	err = db.WithContext(ctx).
		Table("user_plans").
		Select(
			"user_plans.id as user_plan_id",
			"users.username",
			"plans.name as plan_name",
			"resource_types.name as resource_type_name",
			"quotas.quota",
			"usages.usage",
		).
		Joins("JOIN users ON user_plans.user_id = users.id").
		Joins("JOIN plans ON user_plans.plan_id = plans.id").
		Joins("JOIN quotas ON user_plans.id = quotas.user_plan_id").
		Joins("JOIN usages ON user_plans.id = usages.user_plan_id").
		Joins("JOIN resource_types ON usages.resource_type_id = resource_types.id").
		Where("users.username = ?", username).
		Where("resource_types.name = ?", resourceName).
		Where(
			db.Where("CURRENT_TIMESTAMP BETWEEN user_plans.effective_start_date AND user_plans.effective_end_date").
				Or("CURRENT_TIMESTAMP > user_plans.effective_start_date AND user_plans.effective_end_date IS NULL"),
		).
		Where("usages.resource_type_id = quotas.resource_type_id").
		Where("usages.usage >= quotas.quota").
		Find(&result).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed to check for overage")
	}

	k := make([]string, 0)
	for key := range retval {
		k = append(k, key)
	}

	if len(k) > 0 {
		retval["overage"] = true
	} else {
		retval["overage"] = false
	}

	return retval, nil
}
