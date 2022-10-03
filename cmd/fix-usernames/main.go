package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/cyverse/QMS/internal/db"
	"github.com/cyverse/QMS/internal/model"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/env"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Config struct {
	DatabaseURI    string
	UsernameSuffix string
}

// loadConfig loads configuration settings from the environment. We're using koanf directly here so that the
// configuration files don't have to be present to run the configuration utility.
func loadConfig() (*Config, error) {
	k := koanf.New(".")

	// Load the configuration settings from the environment.
	err := k.Load(
		env.Provider("QMS_", ".",
			func(s string) string {
				return strings.Replace(strings.ToLower(strings.TrimPrefix(s, "QMS_")), "_", ".", -1)
			},
		),
		nil,
	)
	if err != nil {
		return nil, err
	}

	// Verify that the database URI is specified.
	databaseURI := k.String("database.uri")
	if databaseURI == "" {
		return nil, fmt.Errorf("QMS_DATABASE_URI must be defined")
	}

	// Verify that the username suffix is specified.
	usernameSuffix := k.String("username.suffix")
	if usernameSuffix == "" {
		return nil, fmt.Errorf("QMS_USERNAME_SUFFIX must be specified")
	}

	return &Config{DatabaseURI: databaseURI, UsernameSuffix: usernameSuffix}, nil
}

// listUsersWithSuffixes lists the users in the database whose usernames contain the username suffix.
func listUsersWithSuffixes(ctx context.Context, tx *gorm.DB, usernameSuffix string) ([]model.User, error) {
	var users []model.User
	err := tx.WithContext(ctx).
		Find(&users, "username like ?", fmt.Sprintf("%%%s", usernameSuffix)).
		Order("username").
		Error
	return users, err
}

// loadCurrentSubscription loads the current subscription for a single user. It does not create a new subscription if
// the user doesn't currently have one.
func loadCurrentSubscription(ctx context.Context, tx *gorm.DB, user *model.User) (*model.UserPlan, error) {
	var subscriptions []model.UserPlan

	// Look up the plan.
	err := tx.WithContext(ctx).
		Preload("User").
		Preload("Plan").
		Preload("Plan.PlanQuotaDefaults").
		Preload("Plan.PlanQuotaDefaults.ResourceType").
		Preload("Quotas").
		Preload("Quotas.ResourceType").
		Preload("Usages").
		Preload("Usages.ResourceType").
		Where("user_id = ?", user.ID).
		Where(
			tx.Where("CURRENT_TIMESTAMP BETWEEN user_plans.effective_start_date AND user_plans.effective_end_date").
				Or("CURRENT_TIMESTAMP > user_plans.effective_start_date AND user_plans.effective_end_date IS NULL"),
		).
		Order("user_plans.effective_start_date desc").
		Limit(1).
		Find(&subscriptions).
		Error

	var plan *model.UserPlan
	if len(subscriptions) > 0 {
		plan = &subscriptions[0]
	}
	return plan, err
}

// LoadSubscription loads the subscription details for the given subscription ID.
func loadSubscription(ctx context.Context, tx *gorm.DB, subscriptionID string) (*model.UserPlan, error) {
	var subscription *model.UserPlan

	err := tx.WithContext(ctx).
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

// loadMostRecentDataUsage loads lastest data usage record for a user using both the username with and without the
// username suffix.
func loadMostRecentDataUsage(ctx context.Context, tx *gorm.DB, oldUsername, newUsername string) (*model.Usage, error) {
	var usages []model.Usage

	// Look up the usages.
	err := tx.WithContext(ctx).
		Joins("JOIN user_plans ON usages.user_plan_id = user_plans.id").
		Joins("JOIN users ON user_plans.user_id = users.id").
		Joins("JOIN resource_types ON usages.resource_type_id = resource_types.id").
		Where("users.username IN ?", []string{oldUsername, newUsername}).
		Where("resource_types.name = ?", "data.size").
		Order("usages.last_modified_at DESC").
		Limit(1).
		Find(&usages).
		Error

	var usage *model.Usage
	if len(usages) > 0 {
		usage = &usages[0]
	}
	return usage, err
}

// findQuotaValue finds the quota value for a specific resource type in a list of quotas.
func findQuotaValue(quotas []model.Quota, resourceTypeName string) float64 {
	var quotaValue float64
	for _, quota := range quotas {
		if quota.ResourceType.Name == resourceTypeName && quota.Quota > quotaValue {
			quotaValue = quota.Quota
		}
	}
	return quotaValue
}

// setQuota either adds a quota to a subscription or updates a quota in a subscription.
func setQuota(ctx context.Context, tx *gorm.DB, subscriptionID, resourceTypeID *string, quotaValue float64) error {
	quota := model.Quota{
		UserPlanID:     subscriptionID,
		ResourceTypeID: resourceTypeID,
		Quota:          quotaValue,
	}
	err := tx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{
					Name: "user_plan_id",
				},
				{
					Name: "resource_type_id",
				},
			},
			DoUpdates: clause.AssignmentColumns([]string{"quota"}),
		}).
		Create(&quota).Error
	return err
}

// restorePreviousQuotas ensures that the resource usage limits for the new subscription are at least as large as the
// resource usage limits for the old subscription.
func restorePreviousQuotas(ctx context.Context, tx *gorm.DB, oldSubscription, newSubscription *model.UserPlan) error {
	for _, quota := range oldSubscription.Quotas {
		newQuotaValue := findQuotaValue(newSubscription.Quotas, quota.ResourceType.Name)
		if newQuotaValue < quota.Quota {
			err := setQuota(ctx, tx, newSubscription.ID, quota.ResourceType.ID, quota.Quota)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// addUsageToSubscription adds a usage record to the new subscription. This function is only intended to be used to
// add usage records to a brand new subscription, so it assumes that there aren't any usages associated with the
// subscription yet.
func addUsageToSubscription(ctx context.Context, tx *gorm.DB, subscription *model.UserPlan, usage *model.Usage) error {
	newUsage := &model.Usage{
		ResourceTypeID: usage.ResourceTypeID,
		UserPlanID:     subscription.ID,
		Usage:          usage.Usage,
	}
	return tx.WithContext(ctx).Create(newUsage).Error
}

// fixUsername fixes a username for a single user.
func fixUsername(ctx context.Context, tx *gorm.DB, oldUser *model.User, usernameSuffix string) error {
	fmt.Printf("fixing the subscriptions for %s...\n", oldUser.Username)

	// Get the information for the correct username.
	newUsername := strings.TrimSuffix(oldUser.Username, usernameSuffix)
	newUser, err := db.GetUser(ctx, tx, newUsername)
	if err != nil {
		return errors.Wrapf(err, "unable to get the user details for %s", newUsername)
	}

	// Load the currently oldSubscription for the user.
	oldSubscription, err := loadCurrentSubscription(ctx, tx, newUser)
	if err != nil {
		return errors.Wrapf(err, "unable to load the current plan for %s", newUser.Username)
	}

	// Load the most recent data usage record for the user.
	usage, err := loadMostRecentDataUsage(ctx, tx, oldUser.Username, newUser.Username)
	if err != nil {
		return errors.Wrapf(
			err,
			"unable to load the most recent data usage for %s and %s",
			oldUser.Username,
			newUser.Username,
		)
	}

	// Deactivate all plans for both the old username and the new username.
	err = db.DeactivateUserPlans(ctx, tx, *oldUser.ID)
	if err != nil {
		return errors.Wrapf(err, "unable to deactivate existing plans for %s", oldUser.Username)
	}
	err = db.DeactivateUserPlans(ctx, tx, *newUser.ID)
	if err != nil {
		return errors.Wrapf(err, "unable to deactivate existing plans for %s", newUser.Username)
	}

	// Subscribe the new user to the default plan if they didn't have a current subscription.
	var newSubscription *model.UserPlan
	if oldSubscription == nil {
		newSubscription, err = db.SubscribeUserToDefaultPlan(ctx, tx, newUser.Username)
		if err != nil {
			return errors.Wrapf(err, "unable to subscribe %s to the default plan", newUser.Username)
		}
	} else {
		plan := &oldSubscription.Plan
		newSubscription, err = db.SubscribeUserToPlan(ctx, tx, newUser, plan)
		if err != nil {
			return errors.Wrapf(err, "unable to subscribe %s to the %s plan", newUser.Username, plan.Name)
		}
	}

	// Get all of the details for the new subscription.
	newSubscription, err = loadSubscription(ctx, tx, *newSubscription.ID)
	if err != nil {
		return errors.Wrapf(err, "unable to load the new subscription details for %s", newUser.Username)
	}

	// Ensure that the new quotas are greater than or equal to the old quotas if applicable.
	if oldSubscription != nil {
		err = restorePreviousQuotas(ctx, tx, oldSubscription, newSubscription)
		if err != nil {
			return errors.Wrapf(err, "unable to restore previous quotas for %s", newUser.Username)
		}
	}

	// Associate the usage with the current subscription.
	if usage != nil {
		err = addUsageToSubscription(ctx, tx, newSubscription, usage)
		if err != nil {
			return errors.Wrapf(err, "unable to add data usage to the new subscription for %s", newUser.Username)
		}
	}

	return nil
}

func main() {

	// Load the configuration.
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("unable to load the configuration: %s", err)
	}

	// Establish the database connection.
	_, gormdb, err := db.Init("postgres", cfg.DatabaseURI)
	if err != nil {
		log.Fatalf("unable to connect to the database: %s", err)
	}

	// Run the actual updates in a transaction.
	err = gormdb.Transaction(func(tx *gorm.DB) error {
		ctx := context.Background()

		// Get the list of usernames with suffixes.
		usersToFix, err := listUsersWithSuffixes(ctx, tx, cfg.UsernameSuffix)
		if err != nil {
			return errors.Wrap(err, "unable to list usernames with suffixes")
		}

		// Just list the usernames for now.
		for _, user := range usersToFix {
			err = fixUsername(ctx, tx, &user, cfg.UsernameSuffix)
			if err != nil {
				return errors.Wrapf(err, "unable to fix the username for %s", user.Username)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
