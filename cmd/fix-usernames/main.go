package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/cyverse/QMS/internal/db"
	"github.com/cyverse/QMS/internal/model"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/env"
	"github.com/pkg/errors"
	"gorm.io/gorm"
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
	err := tx.WithContext(ctx).Find(&users, "username like ?", fmt.Sprintf("%%%s", usernameSuffix)).Error
	return users, err
}

// loadPlans loads the user plans for a single user. All of the user plan details will be included in the
// list of plans.
func loadPlans(ctx context.Context, tx *gorm.DB, user *model.User) ([]model.UserPlan, error) {
	var plans []model.UserPlan

	// Look up the plans.
	err := tx.WithContext(ctx).
		Preload("Plan").
		Preload("Quotas").
		Preload("Quotas.ResourceType").
		Preload("Usages", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("usages.created_at")
		}).
		Preload("Usages.ResourceType").
		Where("user_id = ?", user.ID).
		Find(&plans).
		Error

	return plans, err
}

// printUsages prints the usages in a user plan.
func printUsages(user *model.User, userPlan *model.UserPlan) {
	fmt.Printf("#### Usages for %s\n", user.Username)
	for _, usage := range userPlan.Usages {
		encodedUsage, err := json.MarshalIndent(usage, "", "    ")
		if err == nil {
			fmt.Printf("%s\n", encodedUsage)
		}
	}
	fmt.Println("")
}

// fixUsername fixes a username for a single user.
func fixUsername(ctx context.Context, tx *gorm.DB, oldUser *model.User, usernameSuffix string) error {

	// Get the information for the correct username.
	newUsername := strings.TrimSuffix(oldUser.Username, usernameSuffix)
	newUser, err := db.GetUser(ctx, tx, newUsername)
	if err != nil {
		return errors.Wrapf(err, "unable to get the user details for %s", newUsername)
	}

	// Load the list of plans for both the old user and the new user.
	oldPlans, err := loadPlans(ctx, tx, oldUser)
	if err != nil {
		return errors.Wrapf(err, "unable to list the plans for %s", oldUser.Username)
	}
	newPlans, err := loadPlans(ctx, tx, newUser)
	if err != nil {
		return errors.Wrapf(err, "unable to list the plans for %s", newUser.Username)
	}

	// Print the usages for all of the plans.
	for _, plan := range oldPlans {
		printUsages(oldUser, &plan)
	}
	for _, plan := range newPlans {
		printUsages(newUser, &plan)
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
		for i, user := range usersToFix {
			err = fixUsername(ctx, tx, &user, cfg.UsernameSuffix)
			if err != nil {
				return errors.Wrapf(err, "unable to fix the username for %s", user.Username)
			}
			if i > 5 {
				break
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
