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
	v1 := s.Router.Group("/v1")
	v1.GET("/", s.RootHandler)
	//Plans
	plans := v1.Group("/plans")
	plans.GET("/plans", s.GetAllPlans)
	plans.GET("/plans/:plan_id", s.GetPLansForID)

	//Users
	users := v1.Group("/users")
	users.GET("/", s.GetAllUsers)
	users.GET("/:username/plan", s.GetUserPlanDetails)
	users.GET("/:username/quotas", s.GetUserAllQuotas)
	users.GET("/:username/quotas/:quotaId", s.GetUserQuotaDetails)
	users.GET("/:username/usages", s.GetUserUsageDetails)

	//Admin
	admin := v1.Group("/admin")
	admin.GET("/quotas", s.GetAllActiveQuotas)
	admin.PUT("/quotas/:quotaid", s.UpdateQuota)
	admin.GET("/usages", s.GetAllActiveUsage)
	admin.POST("/usages", s.UpdateUsages)

	//Rersources
	v1.GET("/resources", s.GetAllResources)
}
