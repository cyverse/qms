package controllers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cyverse/qms/internal/db"
	"github.com/cyverse/qms/internal/model"
	"github.com/cyverse/qms/internal/query"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// SubscriptionAdderConfig contains the configuration for a subscription adder.
type SubscriptionAdderConfig struct {
	Log   *logrus.Entry
	Ctx   context.Context
	Force bool
}

// SubscriptionAdder encapsulates the addition of subscriptions with a cached index of subscription plans.
type SubscriptionAdder struct {
	cfg         *SubscriptionAdderConfig
	plansByName map[string]*model.Plan
}

// NewSubscriptionAdder creates a new SubscriptionAdder instance.
func NewSubscriptionAdder(tx *gorm.DB, cfg *SubscriptionAdderConfig) (*SubscriptionAdder, error) {
	plansByName, err := db.GetPlansByName(cfg.Ctx, tx)
	if err != nil {
		err = errors.Wrap(err, "unable to load subscription plan information")
		return nil, err
	}

	subscriptionAdder := &SubscriptionAdder{
		cfg:         cfg,
		plansByName: plansByName,
	}
	return subscriptionAdder, nil
}

// subscriptionError returns an error record indicating that a subscription could not be created.
func (sa *SubscriptionAdder) subscriptionError(username, msg string) *model.SubscriptionResponse {
	return &model.SubscriptionResponse{
		Subscription:  model.Subscription{User: &model.User{Username: username}},
		FailureReason: &msg,
	}
}

// subscriptionErrorf returns an error record indicating that a subscription could not be created. This is just a
// utility function to remove some cumbersome code in AddSubscription.
func (sa *SubscriptionAdder) subscriptionErrorf(username string, f string, args ...any) *model.SubscriptionResponse {
	return sa.subscriptionError(username, fmt.Sprintf(f, args...))
}

// AddSubscription subscribes a user to a subscription plan.
func (sa *SubscriptionAdder) AddSubscription(tx *gorm.DB, req model.SubscriptionRequest) *model.SubscriptionResponse {
	username := req.Username
	planName := req.PlanName
	startDate := req.GetStartDate()
	endDate := req.GetEndDate(startDate)
	paid := req.Paid

	if username == nil || *username == "" {
		return sa.subscriptionError("", "no username provided in request")
	}
	if planName == nil || *planName == "" {
		return sa.subscriptionError(*username, "no plan name provided in request")
	}
	if paid == nil {
		return sa.subscriptionError(*username, "no paid indicator provided in request")
	}
	if !startDate.Before(endDate) {
		return sa.subscriptionError(*username, "start date must precede end date")
	}
	if !endDate.After(time.Now()) {
		return sa.subscriptionError(*username, "end date must be in the future")
	}

	// Look up the plan information.
	plan, ok := sa.plansByName[*planName]
	if !ok || plan == nil {
		return sa.subscriptionErrorf(*username, "plan does not exist: %s", *planName)
	}

	// Add some fields to the logger.
	var log = sa.cfg.Log.WithFields(
		logrus.Fields{
			"username":  *username,
			"planName":  *planName,
			"startDate": startDate,
			"endDate":   endDate,
			"paid":      *paid,
		},
	)

	// Get the user information.
	user, err := db.GetUser(sa.cfg.Ctx, tx, *username)
	if err != nil {
		log.Error(err)
		return sa.subscriptionError(*username, err.Error())
	}

	// Check the current plan if we're supposed to.
	if !sa.cfg.Force {
		activeSubscription, err := db.GetActiveSubscriptionDetailsForDate(sa.cfg.Ctx, tx, *username, startDate)
		if err != nil {
			log.Error(err)
			return sa.subscriptionError(*username, err.Error())
		}

		// Compare the CPU allocations to determine the plan levels to determine if the user gets a new subscription.
		activeCPUAllocation := activeSubscription.Plan.GetDefaultQuotaValue(model.RESOURCE_TYPE_CPU_HOURS)
		newCPUAllocation := plan.GetDefaultQuotaValue(model.RESOURCE_TYPE_CPU_HOURS)
		if newCPUAllocation <= activeCPUAllocation {
			return model.SubscriptionResponseFromSubscription(activeSubscription, false)
		}
	}

	// Ensure that no two subscriptions will be active at the same time..
	err = db.DeactivateSubscriptions(sa.cfg.Ctx, tx, *user.ID, startDate, endDate)
	if err != nil {
		log.Error(err)
		return sa.subscriptionError(*username, err.Error())
	}

	// Add the subscription.
	sub, err := db.SubscribeUserToPlan(sa.cfg.Ctx, tx, user, plan, &req.SubscriptionOptions)
	if err != nil {
		log.Error(err)
		return sa.subscriptionError(*username, err.Error())
	}

	// Load the subscription details.
	sub, err = db.GetSubscriptionDetails(sa.cfg.Ctx, tx, *sub.ID)
	if err != nil {
		log.Error(err)
		return sa.subscriptionError(*username, err.Error())
	}

	return model.SubscriptionResponseFromSubscription(sub, true)
}

// AddSubscriptions creates the subscriptions described in the request body.
//
// swagger:route POST /v1/subscriptions subscriptions addSubscriptions
//
// # Add Subscriptions
//
// Creates the subscriptions described in the request body.
//
// Responses:
//
//	200: subscriptionsResponse
func (s Server) AddSubscriptions(ctx echo.Context) error {
	var err error

	// Initialize the context for the endpoint.
	var log = log.WithField("context", "add-subscriptions")
	var context = ctx.Request().Context()

	// Parse the request body.
	var body model.SubscriptionRequests
	err = ctx.Bind(&body)
	if err != nil {
		msg := fmt.Sprintf("invalid request body: %s", err)
		log.Error(msg)
		return model.Error(ctx, msg, http.StatusBadRequest)
	}

	// Get the value of the `force` query parameter.
	force := true
	force, err = query.ValidateBooleanQueryParam(ctx, "force", &force)
	if err != nil {
		msg := fmt.Sprintf("invalid value for query parameter, force: %s", err)
		log.Error(msg)
		return model.Error(ctx, msg, http.StatusBadRequest)
	}

	// Create a new subscription adder.
	saConfig := &SubscriptionAdderConfig{
		Log:   log,
		Ctx:   context,
		Force: force,
	}
	subscriptionAdder, err := NewSubscriptionAdder(s.GORMDB, saConfig)
	if err != nil {
		log.Error(err)
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}

	// Add a separate subscription for each subscription request in the request body.
	response := make([]*model.SubscriptionResponse, len(body.Subscriptions))
	for i, subscriptionRequest := range body.Subscriptions {
		_ = s.GORMDB.Transaction(func(tx *gorm.DB) error {
			response[i] = subscriptionAdder.AddSubscription(
				tx,
				subscriptionRequest,
			)
			return nil
		})
	}

	return model.Success(ctx, response, http.StatusOK)
}

// ListSubscriptions is the handler for the GET /v1/subscriptions endpoint.
//
// swagger:route GET /v1/subscriptions subscriptions listSubscriptions
//
// # List Subscriptions
//
// Lists existing CyVerse subscriptions.
//
// Responses:
//
//	200: subscriptionListing
func (s Server) ListSubscriptions(ctx echo.Context) error {
	var err error

	// Initialize the context for the endpoint.
	var log = log.WithField("context", "list-subscriptions")
	var context = ctx.Request().Context()

	// Extract the query parameters.
	var offset int32 = 0
	offset, err = query.ValidateIntQueryParam(ctx, "offset", &offset, "gte=0")
	if err != nil {
		log.Error(err)
		return model.Error(ctx, err.Error(), http.StatusBadRequest)
	}
	var limit int32 = 50
	limit, err = query.ValidateIntQueryParam(ctx, "limit", &limit, "gte=0")
	if err != nil {
		log.Error(err)
		return model.Error(ctx, err.Error(), http.StatusBadRequest)
	}
	sortField := "username"
	validSortFields := []string{"username", "start-date", "end-date"}
	sortField, err = query.ValidateEnumQueryParam(ctx, "sort-field", validSortFields, &sortField)
	if err != nil {
		log.Error(err)
		return model.Error(ctx, err.Error(), http.StatusBadRequest)
	}
	sortDir, err := query.ValidateSortDir(ctx)
	if err != nil {
		log.Error(err)
		return model.Error(ctx, err.Error(), http.StatusBadRequest)
	}
	search := ctx.QueryParam("search")

	// Determine the sort field to pass to the database.
	dbSortFieldFor := map[string]string{
		"username":   "users.username",
		"start-date": "subscriptions.effective_start_date",
		"end-date":   "subscriptions.effective_end_date",
	}
	dbSortField, ok := dbSortFieldFor[sortField]
	if !ok {
		err := fmt.Errorf("sort field name inconsistency detected for %s: please contact support", dbSortField)
		log.Error(err)
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}

	// Obtain the subscription listing.
	var subscriptions []*model.Subscription
	var count int64
	err = s.GORMDB.Transaction(func(tx *gorm.DB) error {
		params := &db.SubscriptionListingParams{
			Offset:    int(offset),
			Limit:     int(limit),
			SortField: dbSortField,
			SortDir:   sortDir,
			Search:    search,
		}
		subscriptions, count, err = db.ListSubscriptions(context, tx, params)
		return err
	})
	if err != nil {
		log.Error(err)
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}

	// Build the result.
	return model.Success(
		ctx,
		&model.SubscriptionListing{
			Subscriptions: subscriptions,
			Total:         count,
		},
		http.StatusOK,
	)
}
