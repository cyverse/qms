package controllers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cyverse/QMS/internal/db"
	"github.com/cyverse/QMS/internal/model"
	"github.com/cyverse/QMS/internal/query"
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

// subscriptionError returns an error record indicating that a subscription could not be created. This is just a
// utility function to remove some cumbersome code in AddSubscription.
func (sa *SubscriptionAdder) subscriptionError(f string, args ...any) *model.SubscriptionResponse {
	msg := fmt.Sprintf(f, args...)
	return &model.SubscriptionResponse{FailureReason: &msg}
}

// AddSubscription subscribes a user to a subscription plan.
func (sa *SubscriptionAdder) AddSubscription(tx *gorm.DB, username, planName *string) *model.SubscriptionResponse {
	if username == nil || *username == "" {
		return sa.subscriptionError("no username provied in request")
	}
	if planName == nil || *planName == "" {
		return sa.subscriptionError("no plan name provided in request")
	}

	// Look up the plan information.
	plan, ok := sa.plansByName[*planName]
	if !ok || plan == nil {
		return sa.subscriptionError("plan does not exist: %s", *planName)
	}

	// Add some fields to the logger.
	var log = sa.cfg.Log.WithFields(
		logrus.Fields{
			"username": *username,
			"planName": *planName,
		},
	)

	// Get the user information.
	user, err := db.GetUser(sa.cfg.Ctx, tx, *username)
	if err != nil {
		log.Error(err)
		return sa.subscriptionError(err.Error())
	}

	// Check the current plan if we're supposed to.
	if !sa.cfg.Force {
		activeSubscription, err := db.GetActiveUserPlanDetails(sa.cfg.Ctx, tx, *username)
		if err != nil {
			log.Error(err)
			return sa.subscriptionError(err.Error())
		}

		// Compare the CPU allocations to determine the plan levels to determine if the user gets a new subscription.
		activeCPUAllocation := activeSubscription.Plan.GetDefaultQuotaValue(model.RESOURCE_TYPE_CPU_HOURS)
		newCPUAllocation := plan.GetDefaultQuotaValue(model.RESOURCE_TYPE_CPU_HOURS)
		if newCPUAllocation <= activeCPUAllocation {
			return model.SubscriptionResponseFromUserPlan(activeSubscription, false)
		}
	}

	// Add the subscription.
	sub, err := db.SubscribeUserToPlan(sa.cfg.Ctx, tx, user, plan)
	if err != nil {
		log.Error(err)
		return sa.subscriptionError(err.Error())
	}

	// Load the subscription details.
	sub, err = db.GetUserPlanDetails(sa.cfg.Ctx, tx, *sub.ID)
	if err != nil {
		log.Error(err)
		return sa.subscriptionError(err.Error())
	}

	return model.SubscriptionResponseFromUserPlan(sub, true)
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
				subscriptionRequest.Username,
				subscriptionRequest.PlanName,
			)
			return nil
		})
	}

	return model.Success(ctx, response, http.StatusOK)
}
