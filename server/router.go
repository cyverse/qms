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

func registerUserEndpoints(users *echo.Group, s *controllers.Server) {
	// Lists all of the users.
	users.GET("", s.GetAllUsers)

	// Lists all of the active user plans.
	users.GET("/all_active_users", s.GetAllActiveUserPlans)

	// Updates or adds a quota (read as limit) to a user's current plan.
	users.POST("/quota", s.AddQuota)

	// Adds a new user to the database.
	users.PUT("/:username", s.AddUser)

	// Gets a users's current plan details
	users.GET("/:username/plan", s.GetUserPlanDetails)

	// GET /:username/resources/overages returns summaries of any usages that exceed the quota for the corresponding resource.
	users.GET("/:username/resources/overages", s.GetUserOverages)

	// GET /:username/resources/:resource-name/overage returns whether the usage exceeds the quota for the resource.
	users.GET("/:username/resources/:resource-name/in-overage", s.InOverage)

	// Changes the user's current plan to one corresponding to plan name.
	users.PUT("/:username/:plan_name", s.UpdateUserPlan)
}

func registerPlanEndpoints(plans *echo.Group, s *controllers.Server) {
	// Returns a listing of all available plans
	plans.GET("", s.GetAllPlans)

	// Adds a plan to the database.
	plans.POST("", s.AddPlan)

	// Gets the details of a plan by its UUID.
	plans.GET("/:plan_id", s.GetPlanByID)

	// Adds or updates the quota defaults of a plan.
	plans.POST("/quota-defaults", s.AddPlanQuotaDefault)
}

func registerResourceTypeEndpoints(resourceTypes *echo.Group, s *controllers.Server) {
	// Lists the available resource types.
	resourceTypes.GET("", s.ListResourceTypes)

	// Adds a new resource type to the database
	resourceTypes.POST("", s.AddResourceType)

	// Get the details about a resource type.
	resourceTypes.GET("/:resource_type_id", s.GetResourceTypeDetails)

	// Update details for a resource type.
	resourceTypes.PUT("/:resource_type_id", s.UpdateResourceType)
}

func RegisterHandlers(s controllers.Server) {

	// The base URL acts as a health check endpoint.
	s.Router.GET("/", s.RootHandler)

	// API version 1 endpoints.
	v1 := s.Router.Group("/v1")
	v1.GET("", s.V1RootHandler)

	plans := v1.Group("/plans")
	registerPlanEndpoints(plans, &s)

	subscriptions := v1.Group("/subscriptions")
	subscriptions.POST("", s.AddSubscriptions)
	subscriptions.POST("/", s.AddSubscriptions)

	usages := v1.Group("/usages")
	usages.GET("/:username", s.GetAllUsageOfUser)
	usages.POST("", s.AddUsages)
	usages.GET("/:username/updates", s.GetAllUsageUpdatesForUser)

	users := v1.Group("/users")
	registerUserEndpoints(users, &s)

	resourceTypes := v1.Group("/resource-types")
	registerResourceTypeEndpoints(resourceTypes, &s)

}
