package controllers

import (
	"database/sql"
	"net/http"

	"github.com/cyverse/QMS/internal/model"
	"github.com/labstack/echo"
	"gorm.io/gorm"
)

// API defines the REST API of the notifications service
type Server struct {
	Router  *echo.Echo
	DB      *sql.DB
	GORMDB  *gorm.DB
	Service string
	Title   string
	Version string
}

// RootHandler handles GET requests to the / endpoint.
func (s Server) RootHandler(ctx echo.Context) error {
	resp := model.RootResponse{
		Service: s.Service,
		Title:   s.Title,
		Version: s.Version,
	}
	return ctx.JSON(http.StatusOK, resp)
}
