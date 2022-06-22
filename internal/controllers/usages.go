package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cyverse-de/p/go/qms"
	"github.com/cyverse/QMS/internal/db"
	"github.com/cyverse/QMS/internal/model"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Usage struct {
	Username     string  `json:"username"`
	ResourceName string  `json:"resource_name"`
	UsageValue   float64 `json:"usage_value"`
	UpdateType   string  `json:"update_type"`
}

func (s Server) addUsage(ctx context.Context, usage *Usage) (int, string, error) {
	var (
		err           error
		errStatusCode int
		successMsg    string
	)

	if usage.Username == "" {
		return http.StatusBadRequest, successMsg, errors.New("invalid username")
	}

	if usage.ResourceName == "" {
		return http.StatusBadRequest, successMsg, errors.New("invalid resource name")
	}

	if usage.UsageValue < 0 {
		return http.StatusBadRequest, successMsg, errors.New("invalid usage value")
	}

	if usage.UpdateType == "" {
		return http.StatusBadRequest, successMsg, errors.New("missing usage update type value")
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
		userPlan, err := db.GetActiveUserPlan(ctx, tx, usage.Username)
		if err != nil {
			errStatusCode = http.StatusInternalServerError
			return err
		}

		log.Debugf("active plan is %s", userPlan.Plan.Name)

		// Look up the resource type.
		resourceType, err := db.GetResourceTypeByName(ctx, tx, usage.ResourceName)
		if err != nil {
			errStatusCode = http.StatusInternalServerError
			return err
		}
		if resourceType == nil {
			errStatusCode = http.StatusBadRequest
			return fmt.Errorf("resource type '%s' does not exist", usage.ResourceName)
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
		err = tx.WithContext(ctx).Debug().First(&updateOperation).Error
		if err == gorm.ErrRecordNotFound {
			errStatusCode = http.StatusBadRequest
			return errors.New("invalid update type")
		}
		if err != nil {
			errStatusCode = http.StatusInternalServerError
			return err
		}

		log.Debug("verified update operation from database")

		// Determine the current usage, which should be zero if the usage record doesn't exist.
		currentUsage := model.Usage{
			UserPlanID:     userPlan.ID,
			ResourceTypeID: resourceType.ID,
		}
		err = tx.WithContext(ctx).Debug().First(&currentUsage).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			errStatusCode = http.StatusInternalServerError
			return err
		}

		log.Debugf("got the current usage of %f", currentUsage.Usage)

		// Update the new usage based on the values in the request body.
		switch usage.UpdateType {
		case UpdateTypeSet:
			newUsage.Usage = usage.UsageValue
		case UpdateTypeAdd:
			newUsage.Usage = currentUsage.Usage + usage.UsageValue
		default:
			errStatusCode = http.StatusBadRequest
			return fmt.Errorf("invalid update type: %s", usage.UpdateType)
		}

		log.Debugf("calculated the new usage to be %f", newUsage.Usage)

		// Either add the new usage record or update the existing one.
		err = tx.WithContext(ctx).Debug().Clauses(clause.OnConflict{
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
			errStatusCode = http.StatusInternalServerError
			return err
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
		err = tx.WithContext(ctx).Debug().Create(&update).Error
		if err != nil {
			errStatusCode = http.StatusInternalServerError
			return err
		}

		log.Debug("recorded the update in the databse")

		// Return a response to the caller.
		errStatusCode = http.StatusOK
		successMsg = fmt.Sprintf("successfully updated the usage for: %s", usage.Username)
		return nil
	})

	return errStatusCode, successMsg, err
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

	log.Debug("validated usage information %+v", usage)

	statusCode, msg, err := s.addUsage(context, &usage)
	if statusCode != http.StatusOK {
		if err != nil {
			log.Error(err)
			return model.Error(ctx, err.Error(), statusCode)
		}
		log.Error(msg)
		return model.Error(ctx, msg, statusCode)
	}

	log.Debug("added usage inforamtion %+v", usage)

	return model.SuccessMessage(ctx, msg, statusCode)
}

func (s Server) AddUsagesNATS(subject, reply string, request *qms.AddUsage) {

}

func (s Server) GetAllUsageOfUser(ctx echo.Context) error {
	var err error

	log := log.WithFields(logrus.Fields{"context": "getting all user usages"})

	context := ctx.Request().Context()

	username := ctx.Param("username")
	if username == "" {
		return model.Error(ctx, "invalid username", http.StatusBadRequest)
	}

	log = log.WithFields(logrus.Fields{"user": username})
	log.Debug("got user from request")

	var user model.User
	err = s.GORMDB.WithContext(context).Where("username=?", username).Find(&user).Error
	if err != nil {
		return model.Error(ctx, "user name not found", http.StatusInternalServerError)
	}

	log.Debug("got user from database")

	activePlan, err := db.GetActiveUserPlan(context, s.GORMDB, username)
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}

	log = log.WithFields(logrus.Fields{"activePlan": activePlan.Plan.Name})
	log.Debug("got the active plan for the user from the database")

	var userPlan model.UserPlan
	err = s.GORMDB.WithContext(context).
		Preload("Usages").
		Preload("Usages.ResourceType").
		Where("user_id=?", user.ID).
		Where("plan_id=?", activePlan.PlanID).
		Find(&userPlan).Error

	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}

	log.Debug("got the usages from the database")

	return model.Success(ctx, userPlan.Usages, http.StatusOK)
}

func (s Server) GetUsagesNATS(subject, reply string, request *qms.GetUsages) {

}
