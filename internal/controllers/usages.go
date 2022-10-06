package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cyverse-de/go-mod/gotelnats"
	"github.com/cyverse-de/go-mod/pbinit"
	"github.com/cyverse-de/p/go/qms"
	"github.com/cyverse-de/p/go/svcerror"
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

var (
	ErrUserNotFound        = errors.New("user name not found")
	ErrInvalidUsername     = errors.New("invalid username")
	ErrInvalidResourceName = errors.New("invalid resource name")
	ErrInvalidUsageValue   = errors.New("invalid usage value")
	ErrInvalidUpdateType   = errors.New("invalid update type")
)

func httpStatusCode(err error) int {
	switch err {
	case ErrUserNotFound:
		return http.StatusNotFound
	case ErrInvalidUsername:
		return http.StatusBadRequest
	case ErrInvalidResourceName:
		return http.StatusBadRequest
	case ErrInvalidUsageValue:
		return http.StatusBadRequest
	case ErrInvalidUpdateType:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func natsStatusCode(err error) svcerror.ErrorCode {
	switch err {
	case ErrUserNotFound:
		return svcerror.ErrorCode_NOT_FOUND
	case ErrInvalidUsername:
		return svcerror.ErrorCode_BAD_REQUEST
	case ErrInvalidResourceName:
		return svcerror.ErrorCode_BAD_REQUEST
	case ErrInvalidUsageValue:
		return svcerror.ErrorCode_BAD_REQUEST
	case ErrInvalidUpdateType:
		return svcerror.ErrorCode_BAD_REQUEST
	default:
		return svcerror.ErrorCode_INTERNAL
	}
}

func (s Server) addUsage(ctx context.Context, usage *Usage) error {
	username := strings.TrimSuffix(usage.Username, s.UsernameSuffix)
	if username == "" {
		return ErrInvalidUsername
	}

	if usage.ResourceName == "" {
		return ErrInvalidResourceName
	}

	if usage.UsageValue < 0 {
		return ErrInvalidUsageValue
	}

	if usage.UpdateType == "" {
		return ErrInvalidUpdateType
	}

	log.Debug("validated usage information")

	log = log.WithFields(logrus.Fields{
		"user":       username,
		"resource":   usage.ResourceName,
		"updateType": usage.UpdateType,
		"value":      usage.UsageValue,
	})

	return s.GORMDB.Transaction(func(tx *gorm.DB) error {
		// Look up the currently active user plan, adding a default plan if one doesn't exist already.
		userPlan, err := db.GetActiveUserPlan(ctx, tx, username)
		if err != nil {
			return err
		}

		log.Debugf("active plan is %s", userPlan.Plan.Name)

		// Look up the resource type.
		resourceType, err := db.GetResourceTypeByName(ctx, tx, usage.ResourceName)
		if err != nil {
			return err
		}
		if resourceType == nil {
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
			return errors.New("invalid update type")
		}
		if err != nil {
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
			return err
		}

		log.Debug("recorded the update in the databse")

		return nil
	})
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

	log.Debugf("validated usage information %+v", usage)

	if err = s.addUsage(context, &usage); err != nil {
		log.Error(err)
		return model.Error(ctx, err.Error(), httpStatusCode(err))
	}

	log.Debugf("added usage inforamtion %+v", usage)
	username := strings.TrimSuffix(usage.Username, s.UsernameSuffix)
	successMsg := fmt.Sprintf("successfully updated the usage for: %s", username)

	return model.SuccessMessage(ctx, successMsg, http.StatusOK)
}

func (s Server) AddUsagesNATS(subject, reply string, request *qms.AddUsage) {
	var (
		err   error
		usage Usage
	)

	log := log.WithFields(logrus.Fields{"context": "adding usage information"})

	log.Debugf("subject: %s; reply: %s", subject, reply)

	response := pbinit.NewUsageResponse()
	ctx, span := pbinit.InitAddUsage(request, subject)
	defer span.End()

	username := strings.TrimSuffix(request.Username, s.UsernameSuffix)
	usage = Usage{
		Username:     username,
		ResourceName: request.ResourceName,
		UsageValue:   request.UsageValue,
		UpdateType:   request.UpdateType,
	}

	jsonUsage, err := json.Marshal(usage)
	if err != nil {
		log.Errorf("unable to JSON encode the usage update for %s: %s", username, err.Error())
	} else {
		log.Debugf("received a usage update: %s", jsonUsage)
	}

	if err = s.addUsage(ctx, &usage); err != nil {
		response.Error = gotelnats.InitServiceError(
			ctx, err, &gotelnats.ErrorOptions{
				ErrorCode: natsStatusCode(err),
			},
		)
	} else {
		u := qms.Usage{
			Usage: request.UsageValue,
			ResourceType: &qms.ResourceType{
				Name: request.ResourceName,
			},
		}
		response.Usage = &u
	}

	if reply != "" {
		if err = gotelnats.PublishResponse(ctx, s.NATSConn, reply, response); err != nil {
			log.Error(err)
		}
	} else {
		log.Info("reply subject was empty, not sending response")
	}

}

func (s Server) userUsages(ctx context.Context, username string) (*model.UserPlan, error) {
	var err error

	log = log.WithFields(logrus.Fields{"user": username})
	log.Debug("got user from request")

	var user model.User
	err = s.GORMDB.WithContext(ctx).Where("username=?", username).Find(&user).Error
	if err != nil {
		return nil, errors.New("user name not found")
	}

	log.Debug("got user from database")

	activePlan, err := db.GetActiveUserPlan(ctx, s.GORMDB, username)
	if err != nil {
		return nil, err
	}

	log = log.WithFields(logrus.Fields{"activePlan": activePlan.Plan.Name})
	log.Debug("got the active plan for the user from the database")

	var userPlan model.UserPlan
	err = s.GORMDB.WithContext(ctx).
		Preload("Usages").
		Preload("Usages.ResourceType").
		Where("user_id=?", user.ID).
		Where("plan_id=?", activePlan.PlanID).
		Find(&userPlan).Error

	if err != nil {
		return nil, err
	}

	log.Debug("got the usages from the database")

	return &userPlan, nil
}

func (s Server) GetAllUsageOfUser(ctx echo.Context) error {
	var err error

	log := log.WithFields(logrus.Fields{"context": "getting all user usages"})

	context := ctx.Request().Context()

	username := strings.TrimSuffix(ctx.Param("username"), s.UsernameSuffix)
	if username == "" {
		return model.Error(ctx, "invalid username", http.StatusBadRequest)
	}

	log = log.WithFields(logrus.Fields{"user": username})

	userPlan, err := s.userUsages(context, username)
	if err != nil {
		sCode := httpStatusCode(err)
		log.Error(err)
		return model.Error(ctx, err.Error(), sCode)
	}

	log.Info("successfully found usages")

	return model.Success(ctx, userPlan.Usages, http.StatusOK)
}

func (s Server) GetUsagesNATS(subject, reply string, request *qms.GetUsages) {
	var err error

	log := log.WithFields(logrus.Fields{"context": "getting usages"})
	response := pbinit.NewUsageList()
	ctx, span := pbinit.InitGetUsages(request, subject)
	defer span.End()

	username := strings.TrimSuffix(request.Username, s.UsernameSuffix)
	if username == "" {
		response.Error = gotelnats.InitServiceError(
			ctx, err, &gotelnats.ErrorOptions{
				ErrorCode: svcerror.ErrorCode_BAD_REQUEST,
			},
		)
	}

	log = log.WithFields(logrus.Fields{"user": username})

	userPlan, err := s.userUsages(ctx, username)
	if err != nil {
		log.Error(err)
		response.Error = gotelnats.InitServiceError(
			ctx, err, &gotelnats.ErrorOptions{
				ErrorCode: natsStatusCode(err),
			},
		)
	}

	for _, usage := range userPlan.Usages {
		response.Usages = append(response.Usages, &qms.Usage{
			Uuid:       *usage.ID,
			Usage:      usage.Usage,
			UserPlanId: *usage.UserPlanID,
			ResourceType: &qms.ResourceType{
				Uuid: *usage.ResourceType.ID,
				Name: usage.ResourceType.Name,
				Unit: usage.ResourceType.Unit,
			},
		})
	}

	log.Info("successfully found usages")

	if err = gotelnats.PublishResponse(ctx, s.NATSConn, reply, response); err != nil {
		log.Error(err)
	}
}
