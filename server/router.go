package server

import (
	"github.com/cyverse/QMS/internal/controllers"
	"github.com/cyverse/QMS/internal/log"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

func InitRouter() *echo.Echo {
	// Create the web server.
	e := echo.New()

	// Set a custom logger.
	e.Logger = log.Logger{Entry: log.InitLogger(true)}

	// Add middleware.
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	return e
}

func RegisterHandlers(s controllers.Server) {
	s.Router.GET("/", s.RootHandler)

	// Register the group for API version 1.
	v1Group := s.Router.Group("/v1")
	v1Group.GET("/", s.RootHandler)
	v1Group.GET("plans", s.RootHandler)
}
