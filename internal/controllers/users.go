package controllers

import (
	"fmt"
	"net/http"

	"gorm.io/gorm"

	"github.com/cyverse-de/go-mod/pbinit"
	"github.com/cyverse-de/p/go/qms"
	"github.com/cyverse/QMS/internal/db"
	"github.com/cyverse/QMS/internal/model"
	"github.com/cyverse/QMS/utils"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

const (
	UpdateTypeSet = "SET"
	UpdateTypeAdd = "ADD"
)

// swagger:route GET /v1/users users listUsers
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
	ResourceTypeID *string
}

// GetUserPlanDetails returns information about the currently active plan for the user.
func (s Server) GetUserPlanDetails(ctx echo.Context) error {
	log := log.WithFields(logrus.Fields{"context": "getting active user plan"})

	context := ctx.Request().Context()

	username := utils.RemoveUsernameSuffix(ctx.Param("username"))
	if username == "" {
		return model.Error(ctx, "invalid username", http.StatusBadRequest)
	}

	log = log.WithFields(logrus.Fields{"user": username})

	// Start a transaction.
	return s.GORMDB.Transaction(func(tx *gorm.DB) error {
		var err error

		// Look up or insert the user.
		user, err := db.GetUser(context, tx, username)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		log.Debugf("found user %s in db", user.Username)

		// Look up or create the user plan.
		userPlan, err := db.GetActiveUserPlan(context, tx, user.Username)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		log.Debugf("user plan is %s", userPlan.Plan.Name)

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

		log.Debugf("returning plan ID %s", *result.PlanID)

		// Return the user plan.
		return model.Success(ctx, result, http.StatusOK)
	})
}

// GetUserOverages is the echo handler for listing the resources that a user is
// in overage for.
func (s Server) GetUserOverages(ctx echo.Context) error {
	log := log.WithFields(logrus.Fields{"context": "getting any overages for the user"})

	context := ctx.Request().Context()

	responseList := pbinit.NewOverageList()

	// Skip the remaining logic because QMS is configured to not report overages.
	if !s.ReportOverages {
		return model.ProtobufJSON(ctx, responseList, http.StatusOK)
	}

	username := utils.RemoveUsernameSuffix(ctx.Param("username"))
	if username == "" {
		return model.Error(ctx, "missing username", http.StatusBadRequest)
	}

	log.WithFields(logrus.Fields{"user": username})

	log.Info("looking up any overages")

	log.Debug("before calling db.GetUserOverages()")
	results, err := db.GetUserOverages(context, s.GORMDB, username)
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}
	log.Debug("after calling db.GetUserOverages()")

	for _, r := range results {
		responseList.Overages = append(responseList.Overages, &qms.Overage{
			ResourceName: r["resource_type_name"].(string),
			Quota:        r["quota"].(float32),
			Usage:        r["usage"].(float32),
		})
	}

	return model.ProtobufJSON(ctx, responseList, http.StatusOK)
}

// InOverage is the echo handler for checking if a user is in overage for a
// resource.
func (s Server) InOverage(ctx echo.Context) error {
	log := log.WithFields(logrus.Fields{"context": "checking if a user's usage is an overage"})

	context := ctx.Request().Context()

	response := pbinit.NewIsOverage()

	// Skip the rest of the logic because QMS is configured to not report overages
	if !s.ReportOverages {
		response.IsOverage = false
		return model.ProtobufJSON(ctx, response, http.StatusOK)
	}

	username := utils.RemoveUsernameSuffix(ctx.Param("username"))
	if username == "" {
		return model.Error(ctx, "missing username", http.StatusBadRequest)
	}

	resource := ctx.Param("resource-name")
	if resource == "" {
		return model.Error(ctx, "missing resource name", http.StatusBadRequest)
	}

	log.WithFields(logrus.Fields{"user": username, "resource": resource})

	log.Info("checking if the usage is an overage")

	results, err := db.IsOverage(context, s.GORMDB, username, resource)
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}

	if results != nil {
		response.IsOverage = results["overage"].(bool)
	}

	return model.ProtobufJSON(ctx, response, http.StatusOK)
}

// AddUser adds a new user to the database. This is a no-op if the user already
// exists.
func (s Server) AddUser(ctx echo.Context) error {
	log := log.WithFields(logrus.Fields{"context": "adding user"})

	context := ctx.Request().Context()

	username := utils.RemoveUsernameSuffix(ctx.Param("username"))
	if username == "" {
		return model.Error(ctx, "invalid username", http.StatusBadRequest)
	}

	log.Debugf("user from request is %s", username)

	log = log.WithFields(logrus.Fields{"user": username})

	// Start a transaction.
	return s.GORMDB.Transaction(func(tx *gorm.DB) error {
		var err error

		// Either add the user to the database or look up the existing user
		// information.
		user, err := db.GetUser(context, tx, username)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		log.Debug("found user in the database")

		// GetActiveUserPlan will automatically subscribe the user to the basic
		// plan if not subscribed already.
		_, err = db.GetActiveUserPlan(context, tx, user.Username)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		log.Debug("ensured that user is subscribed to a plan")

		return model.Success(ctx, "Success", http.StatusOK)
	})
}

// UpdateUserPlan subscribes the user to a new plan.
func (s Server) UpdateUserPlan(ctx echo.Context) error {
	log := log.WithFields(logrus.Fields{"context": "updating user plan"})

	context := ctx.Request().Context()

	planName := ctx.Param("plan_name")
	if planName == "" {
		return model.Error(ctx, "invalid plan name", http.StatusBadRequest)
	}

	log.Debugf("plan name from request is %s", planName)

	username := utils.RemoveUsernameSuffix(ctx.Param("username"))
	if username == "" {
		return model.Error(ctx, "invalid username", http.StatusBadRequest)
	}

	log.Debugf("user name from request is %s", username)

	log = log.WithFields(logrus.Fields{
		"user": username,
		"plan": planName,
	})

	// Start a transaction.
	return s.GORMDB.Transaction(func(tx *gorm.DB) error {
		var err error

		// Either add the user to the database or look up the existing user information.
		user, err := db.GetUser(context, tx, username)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		log.Debug("found user in the database")

		// Verify that a plan with the given name exists.
		plan, err := db.GetPlan(context, tx, planName)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}
		if plan == nil {
			msg := fmt.Sprintf("plan name `%s` not found", planName)
			return model.Error(ctx, msg, http.StatusBadRequest)
		}

		log.Debug("verified that plan exists in database")

		// Deactivate all active plans for the user.
		err = db.DeactivateUserPlans(context, tx, *user.ID)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		log.Debug("deactivated all active plans for the user")

		// Subscribe the user to the plan.
		_, err = db.SubscribeUserToPlan(context, tx, user, plan)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		log.Debug("subscribed user to the new plan")

		return model.Success(ctx, "Success", http.StatusOK)
	})
}
