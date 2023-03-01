package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cyverse/QMS/internal/model"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
func SubscribeUserToPlan(ctx context.Context, db *gorm.DB, user *model.User, plan *model.Plan, paid bool) (*model.Subscription, error) {
	wrapMsg := "unable to add user plan"
	var err error

	// Define the user plan.
	effectiveStartDate := time.Now()
	effectiveEndDate := effectiveStartDate.AddDate(1, 0, 0)
	subscription := model.Subscription{
		EffectiveStartDate: &effectiveStartDate,
		EffectiveEndDate:   &effectiveEndDate,
		UserID:             user.ID,
		PlanID:             plan.ID,
		Quotas:             QuotasFromPlan(plan),
		Paid:               paid,
	}
	err = db.WithContext(ctx).Create(&subscription).Error
	if err != nil {
		return nil, errors.Wrap(err, wrapMsg)
	}

	return &subscription, nil
}

// SubscribeUserToDefaultPlan adds the default user plan to the given user.
func SubscribeUserToDefaultPlan(ctx context.Context, db *gorm.DB, username string) (*model.Subscription, error) {
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
	return SubscribeUserToPlan(ctx, db, user, plan, true)
}

// GetActiveSubscription retrieves the user plan record that is currently active for the user. The effective start
// date must be before the current date and the effective end date must either be null or after the current date.
// If multiple active user plans exist, the one with the most recent effective start date is used. If no active
// user plans exist for the user then a new one for the basic plan is created.
func GetActiveSubscription(ctx context.Context, db *gorm.DB, username string) (*model.Subscription, error) {
	wrapMsg := "unable to get the active user plan"
	var err error

	// Look up the currently active user plan, adding a new one if it doesn't exist already.
	var subscription model.Subscription
	err = db.
		WithContext(ctx).
		Table("subscriptions").
		Joins("JOIN users ON subscriptions.user_id=users.id").
		Where("users.username=?", username).
		Where(
			db.Where("CURRENT_TIMESTAMP BETWEEN subscriptions.effective_start_date AND subscriptions.effective_end_date").
				Or("CURRENT_TIMESTAMP > subscriptions.effective_start_date AND subscriptions.effective_end_date IS NULL"),
		).
		Order("subscriptions.effective_start_date desc").
		First(&subscription).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.Wrap(err, wrapMsg)
	} else if err == gorm.ErrRecordNotFound {
		subPtr, err := SubscribeUserToDefaultPlan(ctx, db, username)
		if err != nil {
			return nil, errors.Wrap(err, wrapMsg)
		}
		subscription = *subPtr
	}

	return &subscription, nil
}

// HasActiveSubscription determines whether or not the user currently has an active user plan.
func HasActiveSubscription(ctx context.Context, db *gorm.DB, username string) (bool, error) {
	wrapMsg := "unable to determine whether the user has an active user plan"

	// Determine whether or not the user has an active subscription.
	var count int64
	err := db.
		WithContext(ctx).
		Table("subscriptions").
		Joins("JOIN users ON subscriptions.user_id=users.id").
		Where("users.username=?", username).
		Where(
			db.Where("CURRENT_TIMESTAMP BETWEEN subscriptions.effective_start_date AND subscriptions.effective_end_date").
				Or("CURRENT_TIMESTAMP > subscriptions.effective_start_date AND subscriptions.effective_end_date IS NULL"),
		).
		Count(&count).
		Error
	if err != nil {
		return false, errors.Wrap(err, wrapMsg)
	}

	return count > 0, nil
}

// GetSubscriptionDetails loads the details for the user plan with the given ID from the database. This function assumes
// that the user plan exists.
func GetSubscriptionDetails(ctx context.Context, db *gorm.DB, subscriptionID string) (*model.Subscription, error) {
	var subscription *model.Subscription

	err := db.WithContext(ctx).
		Preload("User").
		Preload("Plan").
		Preload("Plan.PlanQuotaDefaults").
		Preload("Plan.PlanQuotaDefaults.ResourceType").
		Preload("Quotas").
		Preload("Quotas.ResourceType").
		Preload("Usages").
		Preload("Usages.ResourceType").
		Where("id = ?", subscriptionID).
		First(&subscription).
		Error

	return subscription, err
}

// SubscriptionListingParams represents the parameters that can be used to customize a user plan listing.
type SubscriptionListingParams struct {
	Offset    int
	Limit     int
	SortField string
	SortDir   string
	Search    string
}

// ListSubscriptions lists subscriptions for multiple users.
func ListSubscriptions(ctx context.Context, db *gorm.DB, params *SubscriptionListingParams) ([]*model.Subscription, int64, error) {
	var subscriptions []*model.Subscription
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
	if params != nil && params.SortDir != "" {
		order = params.SortDir
	}
	orderBy := fmt.Sprintf("%s %s", sortField, order)

	// Build the base query.
	baseQuery := db.WithContext(ctx).
		Joins("JOIN users ON subscriptions.user_id=users.id").
		Preload("User").
		Preload("Plan").
		Preload("Plan.PlanQuotaDefaults").
		Preload("Plan.PlanQuotaDefaults.ResourceType").
		Preload("Quotas").
		Preload("Quotas.ResourceType").
		Preload("Usages").
		Preload("Usages.ResourceType").
		Where(
			db.Where("CURRENT_TIMESTAMP BETWEEN subscriptions.effective_start_date AND subscriptions.effective_end_date").
				Or("CURRENT_TIMESTAMP > subscriptions.effective_start_date AND subscriptions.effective_end_date IS NULL"),
		)

	// Add the search clause if we're supposed to.
	if params.Search != "" {
		search := strings.ReplaceAll(params.Search, "%", "\\%")
		search = strings.ReplaceAll(search, "_", "\\_")
		baseQuery = baseQuery.Where("users.username LIKE ?", "%"+search+"%")
	}

	// Count the number of items in the result set.
	err := baseQuery.
		Model(&subscriptions).
		Count(&count).Error

	// Look up the result set.
	if err == nil {
		err = baseQuery.
			Offset(offset).
			Limit(limit).
			Order(orderBy).
			Find(&subscriptions).Error
	}

	return subscriptions, count, err
}

// GetActiveSubscriptionDetails retrieves the user plan information that is currently active for the user. The effective
// start date must be before the current date and the effective end date must either be null or after the current date.
// If multiple active user plans exist, the one with the most recent effective start date is used. If no active user
// plans exist for the user then a new one for the basic plan is created. This funciton is like GetActiveSubscription except
// that it also loads all of the user plan details from the database.
func GetActiveSubscriptionDetails(ctx context.Context, db *gorm.DB, username string) (*model.Subscription, error) {
	var err error

	// Get the current user plan.
	subscription, err := GetActiveSubscription(ctx, db, username)
	if err != nil {
		return nil, err
	}

	// Load the user plan details.
	return GetSubscriptionDetails(ctx, db, *subscription.ID)
}

// DeactivateSubscriptions marks all currently active plans for a user as expired. This operation is used when a user
// selects a new plan. This function does not support user plans that become active in the future at this time.
func DeactivateSubscriptions(ctx context.Context, db *gorm.DB, userID string) error {
	wrapMsg := "unable to deactivate active plans for user"
	// Mark currently active user plans as expired.
	err := db.WithContext(ctx).
		Model(&model.Subscription{}).
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

// UpsertQuota updates a quota if a corresponding quota exists in the database. If a corresponding quota does not
// exist, a new quota will be inserted.
func UpsertQuota(ctx context.Context, db *gorm.DB, quota *model.Quota) error {
	wrapMsg := "unable to insert or update the quota"

	// Either insert or update the quota.
	err := db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{
					Name: "subscription_id",
				},
				{
					Name: "resource_type_id",
				},
			},
			DoUpdates: clause.AssignmentColumns([]string{"quota"}),
		}).
		Create(quota).
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
		Table("subscriptions").
		Select(
			"subscriptions.id as subscription_id",
			"users.username",
			"plans.name as plan_name",
			"resource_types.name as resource_type_name",
			"quotas.quota",
			"usages.usage",
		).
		Joins("JOIN users ON subscriptions.user_id = users.id").
		Joins("JOIN plans ON subscriptions.plan_id = plans.id").
		Joins("JOIN quotas ON subscriptions.id = quotas.subscription_id").
		Joins("JOIN usages ON subscriptions.id = usages.subscription_id").
		Joins("JOIN resource_types ON usages.resource_type_id = resource_types.id").
		Where("users.username = ?", username).
		Where(
			db.Where("CURRENT_TIMESTAMP BETWEEN subscriptions.effective_start_date AND subscriptions.effective_end_date").
				Or("CURRENT_TIMESTAMP > subscriptions.effective_start_date AND subscriptions.effective_end_date IS NULL"),
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
		Table("subscriptions").
		Select(
			"subscriptions.id as subscription_id",
			"users.username",
			"plans.name as plan_name",
			"resource_types.name as resource_type_name",
			"quotas.quota",
			"usages.usage",
		).
		Joins("JOIN users ON subscriptions.user_id = users.id").
		Joins("JOIN plans ON subscriptions.plan_id = plans.id").
		Joins("JOIN quotas ON subscriptions.id = quotas.subscription_id").
		Joins("JOIN usages ON subscriptions.id = usages.subscription_id").
		Joins("JOIN resource_types ON usages.resource_type_id = resource_types.id").
		Where("users.username = ?", username).
		Where("resource_types.name = ?", resourceName).
		Where(
			db.Where("CURRENT_TIMESTAMP BETWEEN subscriptions.effective_start_date AND subscriptions.effective_end_date").
				Or("CURRENT_TIMESTAMP > subscriptions.effective_start_date AND subscriptions.effective_end_date IS NULL"),
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
