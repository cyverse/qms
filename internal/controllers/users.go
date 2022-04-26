package controllers

import (
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/cyverse/QMS/internal/db"
	"github.com/cyverse/QMS/internal/model"
	"github.com/labstack/echo/v4"
)

const (
	UpdateTypeSet = "SET"
	UpdateTypeAdd = "ADD"
)

// swagger:route GET /admin/users admin listUsers
//
// List Users
//
// Lists the users registered in the QMS database.
//
// responses:
//   200: userListing
//   500: internalServerErrorResponse

// GetAllUsers lists the users that are currently defined in the database.
func (s Server) GetAllUsers(ctx echo.Context) error {
	var data []model.User
	err := s.GORMDB.Debug().Find(&data).Error
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}
	return ctx.JSON(http.StatusOK, model.SuccessResponse(data, http.StatusOK))
}

const (
	ValueTypeQuotas = "quotas"
	ValueTypeUsages = "usages"
)

type Result struct {
	ID             *string
	UserName       string
	ResourceTypeId *string
}

// GetUserPlanDetails returns information about the currently active plan for the user.
func (s Server) GetUserPlanDetails(ctx echo.Context) error {
	context := ctx.Request().Context()
	username := ctx.Param("username")
	if username == "" {
		return model.Error(ctx, "invalid username", http.StatusBadRequest)
	}

	// Start a transaction.
	return s.GORMDB.Transaction(func(tx *gorm.DB) error {
		var err error

		// Look up or insert the user.
		user, err := db.GetUser(context, tx, username)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		// Look up or create the user plan.
		userPlan, err := db.GetActiveUserPlan(context, tx, user.Username)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		// Retrieve the user plan so that the associations will be loaded.
		result := model.UserPlan{ID: userPlan.ID}
		err = tx.
			WithContext(context).
			Preload("User").
			Preload("Plan").
			Preload("Quotas").
			Preload("Quotas.ResourceType").
			Preload("Usages").
			Preload("Usages.ResourceType").
			First(&result).Error
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		// Return the user plan.
		return model.Success(ctx, result, http.StatusOK)
	})
}

// AddUser adds a new user to the database. This is a no-op if the user already exists.
func (s Server) AddUser(ctx echo.Context) error {
	context := ctx.Request().Context()
	username := ctx.Param("user_name")
	if username == "" {
		return model.Error(ctx, "invalid username", http.StatusBadRequest)
	}

	// Start a transaction.
	return s.GORMDB.Transaction(func(tx *gorm.DB) error {
		var err error

		// Either add the user to the database or look up the existing user information.
		user, err := db.GetUser(context, tx, username)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		// GetActiveUserPlan will automatically subscribed the user to the basic plan if not subscribed already.
		_, err = db.GetActiveUserPlan(context, tx, user.Username)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		return model.Success(ctx, "Success", http.StatusOK)
	})
}

// UpdateUserPlan subscribes the user to a new plan.
func (s Server) UpdateUserPlan(ctx echo.Context) error {
	context := ctx.Request().Context()

	planName := ctx.Param("plan_name")
	if planName == "" {
		return model.Error(ctx, "invalid plan name", http.StatusBadRequest)
	}
	username := ctx.Param("user_name")
	if username == "" {
		return model.Error(ctx, "invalid username", http.StatusBadRequest)
	}

	// Start a transaction.
	return s.GORMDB.Transaction(func(tx *gorm.DB) error {
		var err error

		// Either add the user to the database or look up the existing user information.
		user, err := db.GetUser(context, tx, username)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		// Verify that a plan with the given name exists.
		plan, err := db.GetPlan(context, tx, planName)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}
		if plan == nil {
			msg := fmt.Sprintf("plan name `%s` not found", planName)
			return model.Error(ctx, msg, http.StatusBadRequest)
		}

		// Deactivate all active plans for the user.
		err = db.DeactivateUserPlans(context, tx, *user.ID)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		// Subscribe the user to the plan.
		_, err = db.SubscribeUserToPlan(context, tx, user, plan)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		return model.Success(ctx, "Success", http.StatusOK)
	})
}

type Usage struct {
	Username     string  `json:"username"`
	ResourceName string  `json:"resource_name"`
	UsageValue   float64 `json:"usage_value"`
	UpdateType   string  `json:"update_type"`
}

// AddUsages adds or updates the usage record for a user, plan, and resource type.
func (s Server) AddUsages(ctx echo.Context) error {
	var (
		err   error
		usage Usage
	)
	context := ctx.Request().Context()

	// Extract and validate the request body.
	if err = ctx.Bind(&usage); err != nil {
		return model.Error(ctx, "invalid request body", http.StatusBadRequest)
	}
	if usage.Username == "" {
		return model.Error(ctx, "invalid username", http.StatusBadRequest)
	}
	if usage.ResourceName == "" {
		return model.Error(ctx, "invalid resource name", http.StatusBadRequest)
	}
	if usage.UsageValue < 0 {
		return model.Error(ctx, "invalid usage value", http.StatusBadRequest)
	}
	if usage.UpdateType == "" {
		return model.Error(ctx, "missing usage update type value", http.StatusBadRequest)
	}
	err = s.GORMDB.Transaction(func(tx *gorm.DB) error {
		// Look up the currently active user plan, adding a default plan if one doesn't exist already.
		userPlan, err := db.GetActiveUserPlan(context, tx, usage.Username)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		// Look up the resource type.
		resourceType, err := db.GetResourceTypeByName(context, tx, usage.ResourceName)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}
		if resourceType == nil {
			return model.Error(ctx, fmt.Sprintf("resource type '%s' does not exist", usage.ResourceName), http.StatusBadRequest)
		}

		// Initialize the new usage record.
		var newUsage = model.Usage{
			Usage:          usage.UsageValue,
			UserPlanID:     userPlan.ID,
			ResourceTypeID: resourceType.ID,
		}

		// Verify that the update operation for the given update type exists.
		updateOperation := model.UpdateOperation{Name: usage.UpdateType}
		err = tx.WithContext(context).Debug().First(&updateOperation).Error
		if err == gorm.ErrRecordNotFound {
			return model.Error(ctx, "invalid update type", http.StatusBadRequest)
		}
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		// Determine the current usage, which should be zero if the usage record doesn't exist.
		currentUsage := model.Usage{
			UserPlanID:     userPlan.ID,
			ResourceTypeID: resourceType.ID,
		}
		err = tx.WithContext(context).Debug().First(&currentUsage).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		// Update the new usage based on the values in the request body.
		switch usage.UpdateType {
		case UpdateTypeSet:
			newUsage.Usage = usage.UsageValue
		case UpdateTypeAdd:
			newUsage.Usage = currentUsage.Usage + usage.UsageValue
		default:
			msg := fmt.Sprintf("invalid update type: %s", usage.UpdateType)
			return model.Error(ctx, msg, http.StatusBadRequest)
		}

		// Either add the new usage record or update the existing one.
		err = tx.WithContext(context).Debug().Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{
					Name: "user_plan_id",
				},
				{
					Name: "resource_type_id",
				},
			},
			UpdateAll: true,
		}).Create(&newUsage).Error
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		// Record the update in the database.
		update := model.Update{
			Value:             newUsage.Usage,
			ValueType:         ValueTypeUsages,
			ResourceTypeID:    resourceType.ID,
			EffectiveDate:     time.Now(),
			UpdateOperationID: updateOperation.ID,
		}
		err = tx.WithContext(context).Debug().Create(&update).Error
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		// Return a response to the caller.
		msg := fmt.Sprintf("successfully updated the usage for: %s", usage.Username)
		return model.SuccessMessage(ctx, msg, http.StatusOK)
	})
	return err
}
