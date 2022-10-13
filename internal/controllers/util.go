package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/cyverse-de/go-mod/gotelnats"
	"github.com/cyverse-de/p/go/svcerror"
	"github.com/cyverse/QMS/internal/db"
	"github.com/cyverse/QMS/internal/model"
	"github.com/labstack/echo/v4"
)

// ValidateUser determines whether or not a username exists in the database. If an error occurs during the lookup or
// the user doesn't exist then the appropriate response will be sent to the caller and an error will be returned.
func (s Server) ValidateUser(ctx echo.Context, username string) error {
	exists, err := db.UserExists(ctx.Request().Context(), s.GORMDB, username)
	if err != nil {
		sendErr := model.Error(ctx, err.Error(), http.StatusInternalServerError)
		if sendErr != nil {
			ctx.Logger().Errorf("unable to send response: %s", sendErr.Error())
		}
		return err
	}
	if !exists {
		msg := fmt.Sprintf("user %s does not exist", username)
		sendErr := model.Error(ctx, msg, http.StatusNotFound)
		if sendErr != nil {
			ctx.Logger().Errorf("unable to send response: %s", sendErr.Error())
		}
		return errors.New(msg)
	}
	return nil
}

func natsError(ctx context.Context, err error) *svcerror.ServiceError {
	return gotelnats.InitServiceError(
		ctx, err, &gotelnats.ErrorOptions{
			ErrorCode: natsStatusCode(err),
		},
	)
}
