package controllers

import (
	"net/http"

	"github.com/cyverse/QMS/internal/model"
	"github.com/labstack/echo"
)

func (s Server) GetAllPlans(ctx echo.Context) error {
	data := []model.Plans{}
	err := s.GORMDB.Debug().Find(&data).Error
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, model.ErrorResponse(err.Error(), http.StatusInternalServerError))
	}
	return ctx.JSON(http.StatusOK, model.SuccessReponse(data, http.StatusOK))
}

func (s Server) GetPLansForID(ctx echo.Context) error {
	plan_id := ctx.Param("plan_id")
	data := model.Plans{ID: plan_id}
	err := s.GORMDB.Debug().Find(&data).Error
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, model.ErrorResponse(err.Error(), http.StatusInternalServerError))
	}
	return ctx.JSON(http.StatusOK, model.SuccessReponse(data, http.StatusOK))
}
