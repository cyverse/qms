package controllers

import (
	"net/http"

	"github.com/cyverse/QMS/internal/model"
	"github.com/labstack/echo"
)

func (s Server) GetAllResources(ctx echo.Context) error {
	data := []model.ResourceTypes{}
	err := s.GORMDB.Debug().Find(&data).Error
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}
	return ctx.JSON(http.StatusOK, data)
}
