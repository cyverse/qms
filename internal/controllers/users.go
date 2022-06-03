package controllers

import (
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/cyverse-de/go-mod/gotelnats"
	"github.com/cyverse-de/p/go/qms"
	"github.com/cyverse-de/p/go/svcerror"
	"github.com/cyverse/QMS/internal/db"
	"github.com/cyverse/QMS/internal/model"
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

	username := ctx.Param("username")
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

	username := ctx.Param("username")
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

	return model.Success(ctx, results, http.StatusOK)
}

// ListOverages is the NATS handler for listing all of the resources that a user
// is in overage for.
func (s Server) ListOverages(subject, reply string, request *qms.AllUserOveragesRequest) {
	var err error

	log := log.WithFields(logrus.Fields{"context": "list overages"})

	responseList := &qms.OverageList{
		Header:   gotelnats.NewHeader(),
		Overages: make([]*qms.Overage, 0),
	}

	if request.Header == nil {
		request.Header = gotelnats.NewHeader()
	}

	carrier := gotelnats.PBTextMapCarrier{
		Header: request.Header,
	}

	ctx, span := gotelnats.StartSpan(&carrier, subject, gotelnats.Process)
	defer span.End()

	username := request.Username

	results, err := db.GetUserOverages(ctx, s.GORMDB, username)
	if err != nil {
		responseList.Error = gotelnats.InitServiceError(
			ctx, err, &gotelnats.ErrorOptions{
				ErrorCode: svcerror.ErrorCode_INTERNAL,
			},
		)

	}
	log.Debug("after calling db.GetUserOverages()")

	if results != nil {
		for _, r := range results {
			responseList.Overages = append(responseList.Overages, &qms.Overage{
				ResourceName: r["resource_type_name"].(string),
				Quota:        r["quota"].(float32),
				Usage:        r["usage"].(float32),
			})
		}
	}

	if err = gotelnats.PublishResponse(ctx, s.NATSConn, reply, responseList); err != nil {
		log.Error(err)
	}
}

// InResourceOverage is the NATS handler for checking if a user is in overage
// for a particular resource type.
func (s Server) InResourceOverage(subject, reply string, request *qms.IsOverageRequest) {
	var err error

	log := log.WithFields(logrus.Fields{"context": "check if in overage"})

	response := &qms.IsOverage{
		Header:    gotelnats.NewHeader(),
		IsOverage: false,
	}

	if request.Header == nil {
		request.Header = gotelnats.NewHeader()
	}

	carrier := gotelnats.PBTextMapCarrier{
		Header: request.Header,
	}

	ctx, span := gotelnats.StartSpan(&carrier, subject, gotelnats.Process)
	defer span.End()

	results, err := db.IsOverage(ctx, s.GORMDB, request.GetUsername(), request.GetResourceName())
	if err != nil {
		response.Error = gotelnats.InitServiceError(
			ctx, err, &gotelnats.ErrorOptions{
				ErrorCode: svcerror.ErrorCode_INTERNAL,
			},
		)
	}
	log.Debug("after calling db.IsOverage()")

	log.Debugf("results are %+v\n", results)

	if results != nil {
		response.IsOverage = results["overage"].(bool)
	}

	log.Debugf("response are %+v\n", response.IsOverage)

	if err = gotelnats.PublishResponse(ctx, s.NATSConn, reply, response); err != nil {
		log.Error(err)
	}
}

// IsOverage is the echo handler for checking if a user is in overage for a
// resource.
func (s Server) IsOverage(ctx echo.Context) error {
	log := log.WithFields(logrus.Fields{"context": "checking if a user's usage is an overage"})

	context := ctx.Request().Context()

	username := ctx.Param("username")
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

	return model.Success(ctx, results, http.StatusOK)

}

// AddUser adds a new user to the database. This is a no-op if the user already
// exists.
func (s Server) AddUser(ctx echo.Context) error {
	log := log.WithFields(logrus.Fields{"context": "adding user"})

	context := ctx.Request().Context()

	username := ctx.Param("username")
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

	username := ctx.Param("username")
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

	log := log.WithFields(logrus.Fields{"context": "adding usage information"})

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

	log.Debug("validated usage information")

	log = log.WithFields(logrus.Fields{
		"user":       usage.Username,
		"resource":   usage.ResourceName,
		"updateType": usage.UpdateType,
		"value":      usage.UsageValue,
	})

	err = s.GORMDB.Transaction(func(tx *gorm.DB) error {
		// Look up the currently active user plan, adding a default plan if one doesn't exist already.
		userPlan, err := db.GetActiveUserPlan(context, tx, usage.Username)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		log.Debugf("active plan is %s", userPlan.Plan.Name)

		// Look up the resource type.
		resourceType, err := db.GetResourceTypeByName(context, tx, usage.ResourceName)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}
		if resourceType == nil {
			return model.Error(ctx, fmt.Sprintf("resource type '%s' does not exist", usage.ResourceName), http.StatusBadRequest)
		}

		log.Debug("found resource type in database")

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

		log.Debug("verified update operation from database")

		// Determine the current usage, which should be zero if the usage record doesn't exist.
		currentUsage := model.Usage{
			UserPlanID:     userPlan.ID,
			ResourceTypeID: resourceType.ID,
		}
		err = tx.WithContext(context).Debug().First(&currentUsage).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		log.Debugf("got the current usage of %f", currentUsage.Usage)

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

		log.Debugf("calculated the new usage to be %f", newUsage.Usage)

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

		log.Debug("added/updated the usage record in the database")

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

		log.Debug("recorded the update in the databse")

		// Return a response to the caller.
		msg := fmt.Sprintf("successfully updated the usage for: %s", usage.Username)
		return model.SuccessMessage(ctx, msg, http.StatusOK)
	})
	return err
}
