package controllers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/cyverse-de/go-mod/gotelnats"
	"github.com/cyverse-de/go-mod/pbinit"
	"github.com/cyverse-de/p/go/qms"
	"github.com/cyverse-de/p/go/svcerror"
	"github.com/cyverse/QMS/internal/model"
	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func (s Server) userUpdates(ctx context.Context, username string) ([]model.Update, error) {
	var err error

	updates := make([]model.Update, 0)
	err = s.GORMDB.WithContext(ctx).Debug().
		Table("updates").
		Joins("JOIN users ON updates.user_id = users.id").
		Joins("JOIN update_operations ON updates.update_operation_id = update_operations.id").
		Preload("ResourceType").
		Preload("User").
		Preload("UpdateOperation").
		Where("users.username = ?", username).
		Find(&updates).Error
	if err != nil {
		return nil, err
	}
	return updates, nil
}

func (s Server) addUserUpdate(ctx context.Context, update *model.Update) (*model.Update, error) {
	result := s.GORMDB.WithContext(ctx).Create(update)
	err := result.Error
	return update, err
}

func (s Server) getUserID(ctx context.Context, username string) (string, error) {
	var (
		err error
		id  string
	)
	user := model.User{
		Username: username,
	}
	err = s.GORMDB.WithContext(ctx).Take(&user).Error
	id = *user.ID
	return id, err
}

func (s Server) getResourceTypeID(ctx context.Context, name, unit string) (string, error) {
	var (
		err error
		id  string
		rt  model.ResourceType
	)
	rt = model.ResourceType{
		Name: name,
		Unit: unit,
	}
	err = s.GORMDB.WithContext(ctx).Take(&rt).Error
	id = *rt.ID
	return id, err
}

func (s Server) getOperationID(ctx context.Context, name string) (string, error) {
	var (
		err error
		id  string
		op  model.UpdateOperation
	)
	op = model.UpdateOperation{
		Name: name,
	}
	err = s.GORMDB.WithContext(ctx).Take(&op).Error
	id = *op.ID
	return id, err
}

func (s Server) GetAllUpdatesForUser(subject, reply string, request *qms.UpdateListRequest) {
	var err error

	log := log.WithFields(logrus.Fields{"context": "get all user updates over nats"})
	response := pbinit.NewQMSUpdateListResponse()
	ctx, span := pbinit.InitQMSUpdateListRequest(request, subject)
	defer span.End()

	username := request.User.Username
	if username == "" {
		response.Error = gotelnats.InitServiceError(
			ctx, err, &gotelnats.ErrorOptions{
				ErrorCode: svcerror.ErrorCode_BAD_REQUEST,
			},
		)
	}

	log = log.WithFields(logrus.Fields{"user": username})
	mUpdates, err := s.userUpdates(ctx, username)
	if err != nil {
		log.Error(err)
		response.Error = gotelnats.InitServiceError(
			ctx, err, &gotelnats.ErrorOptions{
				ErrorCode: natsStatusCode(err),
			},
		)
	}
	for _, mu := range mUpdates {
		response.Updates = append(response.Updates, &qms.Update{
			Uuid:          *mu.ID,
			EffectiveDate: timestamppb.New(mu.EffectiveDate),
			ValueType:     mu.ValueType,
			Value:         mu.Value,
			ResourceType: &qms.ResourceType{
				Uuid: *mu.ResourceTypeID,
				Name: mu.ResourceType.Name,
				Unit: mu.ResourceType.Unit,
			},
			Operation: &qms.UpdateOperation{
				Uuid: *mu.UpdateOperation.ID,
				Name: mu.UpdateOperation.Name,
			},
			User: &qms.QMSUser{
				Uuid:     *mu.User.ID,
				Username: mu.User.Username,
			},
		})
	}

	if err = gotelnats.PublishResponse(ctx, s.NATSConn, reply, response); err != nil {
		log.Error(err)
	}
}

func (s Server) validateUpdate(request *qms.AddUpdateRequest) (string, error) {
	username := strings.TrimSuffix(request.Update.User.Username, s.UsernameSuffix)
	if username == "" {
		return "", ErrInvalidUsername
	}

	if request.Update.ResourceType.Name == "" || !lo.Contains(
		model.ResourceTypeNames,
		request.Update.ResourceType.Name,
	) {
		return username, ErrInvalidResourceName
	}

	if request.Update.ResourceType.Unit == "" || !lo.Contains(
		model.ResourceTypeUnits,
		request.Update.ResourceType.Unit,
	) {
		return username, ErrInvalidResourceUnit
	}

	if request.Update.Operation.Name == "" || !lo.Contains(
		model.UpdateOperationNames,
		request.Update.Operation.Name,
	) {
		return username, ErrInvalidOperationName
	}

	if request.Update.ValueType == "" || !lo.Contains(
		[]string{model.UsagesTrackedMetric, model.QuotasTrackedMetric},
		request.Update.ValueType,
	) {
		return username, ErrInvalidValueType
	}

	if request.Update.EffectiveDate == nil {
		return username, ErrInvalidEffectiveDate
	}

	return username, nil
}

func (s Server) AddUpdateForUser(subject, reply string, request *qms.AddUpdateRequest) {
	var (
		err                                 error
		userID, resourceTypeID, operationID string
		update                              *model.Update
	)

	// Initialize the response.
	log := log.WithFields(logrus.Fields{"context": "add a user update over nats"})
	response := pbinit.NewQMSAddUpdateResponse()
	ctx, span := pbinit.InitQMSAddUpdateRequest(request, subject)
	defer span.End()

	// Avoid duplicating a lot of error reporting code.
	sendError := func(ctx context.Context, response *qms.AddUpdateResponse, err error) {
		response.Error = natsError(ctx, err)
		if err = gotelnats.PublishResponse(ctx, s.NATSConn, reply, response); err != nil {
			log.Error(err)
		}
	}

	username, err := s.validateUpdate(request)
	if err != nil {
		sendError(ctx, response, err)
		return
	}

	log = log.WithFields(logrus.Fields{"user": username})

	// Get the userID if it's not provided
	if request.Update.User.Uuid == "" {
		userID, err = s.getUserID(ctx, username)
		if err != nil {
			sendError(ctx, response, err)
			return
		}
	} else {
		userID = request.Update.User.Uuid
	}

	// Get the resource type id if it's not provided.
	if request.Update.ResourceType.Uuid == "" {
		resourceTypeID, err = s.getResourceTypeID(
			ctx,
			request.Update.ResourceType.Name,
			request.Update.ResourceType.Unit,
		)
		if err != nil {
			sendError(ctx, response, err)
			return
		}
	} else {
		resourceTypeID = request.Update.ResourceType.Uuid
	}

	// Get the operation id if it's not provided.
	if request.Update.Operation.Uuid == "" {
		operationID, err = s.getOperationID(
			ctx,
			request.Update.Operation.Name,
		)
		if err != nil {
			sendError(ctx, response, err)
			return
		}
	} else {
		operationID = request.Update.Operation.Uuid
	}

	// construct the model.Update
	if response.Error == nil {
		mUpdate := &model.Update{
			ValueType:      request.Update.ValueType,
			Value:          request.Update.Value,
			EffectiveDate:  request.Update.EffectiveDate.AsTime(),
			ResourceTypeID: &resourceTypeID,
			ResourceType: model.ResourceType{
				ID:   &resourceTypeID,
				Name: request.Update.ResourceType.Name,
				Unit: request.Update.ResourceType.Unit,
			},
			UpdateOperationID: &operationID,
			UserID:            &userID,
			User: model.User{
				ID:       &userID,
				Username: username,
			},
		}

		// Add the update to the database.
		update, err = s.addUserUpdate(ctx, mUpdate)
		if err != nil {
			sendError(ctx, response, err)
			return
		}

		// Use update to adjust the appropriate usage/quota total in the user's
		// current plan.
		if err = s.GORMDB.Transaction(func(tx *gorm.DB) error {
			var err error
			switch update.ValueType {
			case model.UsagesTrackedMetric:
				newUsage := &Usage{
					Username:     username,
					ResourceName: update.ResourceType.Name,
					UpdateType:   update.UpdateOperation.Name,
					UsageValue:   update.Value,
				}
				if err = s.addUsagesNoTX(ctx, newUsage, tx); err != nil {
					return err
				}
			case model.QuotasTrackedMetric:
				newQuota := &QuotaReq{
					Username:     username,
					ResourceName: update.ResourceType.Name,
					QuotaValue:   update.Value,
				}
				if err = s.upsertQuota(ctx, newQuota, tx); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unknown value type in update: %s", update.ValueType)
			}

			return nil
		}); err != nil {
			sendError(ctx, response, err)
			return
		}

		// Set up the object for the response.
		rUpdate := qms.Update{
			Uuid:      *update.ID,
			ValueType: update.ValueType,
			Value:     update.Value,
			ResourceType: &qms.ResourceType{
				Uuid: *update.ResourceTypeID,
				Name: update.ResourceType.Name,
				Unit: update.ResourceType.Unit,
			},
			EffectiveDate: timestamppb.New(update.EffectiveDate),
			Operation: &qms.UpdateOperation{
				Uuid: *update.UpdateOperationID,
			},
			User: &qms.QMSUser{
				Uuid:     *update.UserID,
				Username: update.User.Username,
			},
		}

		response.Update = &rUpdate
	}

	// Send the response to the caller
	if err = gotelnats.PublishResponse(ctx, s.NATSConn, reply, response); err != nil {
		log.Error(err)
	}

}

func (s Server) GetAllUsageUpdatesForUser(ctx echo.Context) error {
	var err error

	log := log.WithFields(logrus.Fields{"context": "get all user updates"})

	username := strings.TrimSuffix(ctx.Param("username"), s.UsernameSuffix)
	if username == "" {
		return model.Error(ctx, "invalid username", http.StatusBadRequest)
	}
	log.WithFields(logrus.Fields{"user": username})

	err = s.ValidateUser(ctx, username)
	if err != nil {
		return nil
	}

	context := ctx.Request().Context()
	updates, err := s.userUpdates(context, username)
	if err != nil {
		sCode := httpStatusCode(err)
		log.Error(err)
		return model.Error(ctx, err.Error(), sCode)
	}

	log.Info("successfully found updates")
	return model.Success(ctx, updates, http.StatusOK)
}
