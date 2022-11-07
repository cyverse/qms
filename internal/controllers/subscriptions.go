package controllers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cyverse/QMS/internal/db"
	"github.com/cyverse/QMS/internal/model"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// SubscriptionAdder encapsulates the addition of subscriptions with a cached index of subscription plans.
type SubscriptionAdder struct {
	log         *logrus.Entry
	ctx         context.Context
	plansByName map[string]*model.Plan
}

// NewSubscriptionAdder creates a new SubscriptionAdder instance.
func NewSubscriptionAdder(log *logrus.Entry, ctx context.Context, tx *gorm.DB) (*SubscriptionAdder, error) {
	plansByName, err := db.GetPlansByName(ctx, tx)
	if err != nil {
		err = errors.Wrap(err, "unable to load subscription plan information")
		return nil, err
	}

	subscriptionAdder := &SubscriptionAdder{
		log:         log,
		ctx:         ctx,
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
	var log = sa.log.WithFields(
		logrus.Fields{
			"username": *username,
			"planName": *planName,
		},
	)
	log.Tracef("adding a new subscription")

	// Get the user information.
	user, err := db.GetUser(sa.ctx, tx, *username)
	if err != nil {
		log.Error(err)
		return sa.subscriptionError(err.Error())
	}

	// Add the subscription.
	sub, err := db.SubscribeUserToPlan(sa.ctx, tx, user, plan)
	if err != nil {
		log.Error(err)
		return sa.subscriptionError(err.Error())
	}

	// Load the subscription details.
	sub, err = db.GetUserPlanDetails(sa.ctx, tx, *sub.ID)
	if err != nil {
		log.Error(err)
		return sa.subscriptionError(err.Error())
	}

	return model.SubscriptionResponseFromUserPlan(sub)
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

	// Create a new subscription adder.
	subscriptionAdder, err := NewSubscriptionAdder(log, context, s.GORMDB)
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
