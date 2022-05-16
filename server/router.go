package server

import (
	"github.com/cyverse-de/echo-middleware/v2/redoc"
	"github.com/cyverse/QMS/internal/controllers"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	echolog "github.com/spirosoik/echo-logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

func InitRouter() *echo.Echo {
	log := log.WithFields(logrus.Fields{"context": "router"})

	// Create the web server.
	e := echo.New()

	// Set a custom logger.
	echoLogger := echolog.NewLoggerMiddleware(log)
	e.Logger = echoLogger

	// Add middleware.
	e.Use(otelecho.Middleware("QMS"))
	e.Use(echoLogger.Hook())
	e.Use(middleware.Recover())
	e.Use(redoc.Serve(redoc.Opts{Title: "CyVerse Quota Management System"}))

	return e
}

func RegisterHandlers(s controllers.Server) {

	// The base URL acts as a health check endpoint.
	s.Router.GET("/", s.RootHandler)

	// API version 1 endpoints.
	v1 := s.Router.Group("/v1")
	v1.GET("", s.V1RootHandler)

	// Plans.
	plans := v1.Group("/plans")
	plans.GET("", s.GetAllPlans)
	plans.POST("", s.AddPlan)
	plans.GET("/:plan_id", s.GetPlanByID)

	usages := v1.Group("/usages")
	usages.GET("/:username", s.GetAllUsageOfUser)
	usages.POST("", s.AddUsages)

	// Users.
	users := v1.Group("/users")
	users.GET("/:username/plan", s.GetUserPlanDetails)

	users.GET("", s.GetAllUsers)
	users.PUT("/:user_name", s.AddUser)
	users.PUT("/:user_name/:plan_name", s.UpdateUserPlan)
	users.POST("/quota", s.AddQuota)
	users.GET("/all_active_users", s.GetAllActiveUserPlans)

	resourceTypes := v1.Group("/resource-types")
	resourceTypes.GET("", s.ListResourceTypes)
	resourceTypes.POST("", s.AddResourceType)
	resourceTypes.GET("/:resource_type_id", s.GetResourceTypeDetails)
	resourceTypes.PUT("/:resource_type_id", s.UpdateResourceType)

	plans.POST("/quota-defaults", s.AddPlanQuotaDefault)
}
