package controllers

import (
	"context"
	"net/http"
	"strings"

	"github.com/cyverse-de/go-mod/gotelnats"
	"github.com/cyverse-de/go-mod/pbinit"
	"github.com/cyverse-de/p/go/qms"
	"github.com/cyverse-de/p/go/svcerror"
	"github.com/cyverse/QMS/internal/model"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s Server) userUpdates(ctx context.Context, username string) ([]model.Update, error) {
	var err error

	updates := make([]model.Update, 0)
	err = s.GORMDB.WithContext(ctx).Debug().
		Table("updates").
		Joins("JOIN users ON updates.user_id = users.id").
		Preload("ResourceType").
		Preload("User").
		Where("users.username = ?", username).
		Find(&updates).Error
	if err != nil {
		return nil, err
	}
	return updates, nil
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
			EffectiveDate: timestamppb.New(mu.EffectiveDate),
			ValueType:     mu.ValueType,
			Value:         mu.Value,
			ResourceType: &qms.ResourceType{
				Uuid: *mu.ResourceTypeID,
				Name: mu.ResourceType.Name,
				Unit: mu.ResourceType.Unit,
			},
			Operation: &qms.UpdateOperation{
				Uuid: *mu.UpdateOperationID,
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
