package controllers

import (
	"fmt"
	"net/http"

	"github.com/cyverse-de/echo-middleware/v2/params"
	"github.com/cyverse/qms/internal/db"
	"github.com/cyverse/qms/internal/httpmodel"
	"github.com/cyverse/qms/internal/model"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// extractPlanID extracts and validates the plan ID path parameter.
func extractPlanID(ctx echo.Context) (string, error) {
	planID, err := params.ValidatedPathParam(ctx, "plan_id", "uuid_rfc4122")
	if err != nil {
		return "", fmt.Errorf("the plan ID must be a valid UUID")
	}
	return planID, nil
}

// GetAllPlans is the handler for the GET /v1/plans endpoint.
//
// swagger:route GET /v1/plans plans listPlans
//
// # List Plans
//
// Lists all the plans that are currently available.
//
// responses:
//
//	200: plansResponse
//	400: badRequestResponse
//	500: internalServerErrorResponse
func (s Server) GetAllPlans(ctx echo.Context) error {
	log := log.WithFields(logrus.Fields{"context": "getting all plans"})

	context := ctx.Request().Context()

	plans, err := db.ListPlans(context, s.GORMDB)
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}

	log.Debug("listing plans from the database")

	return model.Success(ctx, plans, http.StatusOK)
}

// GetPlanByID returns the plan with the given identifier.
//
// swagger:route GET /plans/{plan_id} plans getPlanByID
//
// # Get Plan Information
//
// Returns the plan with the given identifier.
//
// responses:
//
//	200: planResponse
//	400: badRequestResponse
//	500: internalServerErrorResponse
func (s Server) GetPlanByID(ctx echo.Context) error {
	var err error

	log := log.WithFields(logrus.Fields{"context": "getting plan by id"})

	context := ctx.Request().Context()

	// Extract and validate the plan ID.
	planID, err := extractPlanID(ctx)
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusBadRequest)
	}

	log = log.WithFields(logrus.Fields{"planID": planID})
	log.Debug("extracted and validated then plan ID from request")

	// Look up the plan.
	plan, err := db.GetPlanByID(context, s.GORMDB, planID)
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}
	if plan == nil {
		msg := fmt.Sprintf("plan ID %s not found", planID)
		return model.Error(ctx, msg, http.StatusNotFound)
	}

	log.Debug("successfully looked up plan to return")

	return model.Success(ctx, plan, http.StatusOK)
}

// AddPlan adds a new plan to the database.
//
// swagger:route POST /plans plans addPlan
//
// # Add Plan
//
// Adds the plan to the Plans Database.
//
// Responses:
//
//	200: successMessageResponse
//	400: badRequestResponse
//	409: conflictResponse
//	500: internalServerErrorResponse
func (s Server) AddPlan(ctx echo.Context) error {
	var err error

	log := log.WithFields(logrus.Fields{"context": "adding plan"})

	context := ctx.Request().Context()

	// Parse and validate the request body.
	var plan httpmodel.NewPlan
	if err = ctx.Bind(&plan); err != nil {
		return model.Error(ctx, err.Error(), http.StatusBadRequest)
	}
	if err = plan.Validate(); err != nil {
		return model.Error(ctx, err.Error(), http.StatusBadRequest)
	}

	log = log.WithFields(logrus.Fields{"plan": plan.Name})
	log.Debugf("adding a new plan to the database: %+v", plan)

	// Begin a transaction.
	return s.GORMDB.Transaction(func(tx *gorm.DB) error {
		dbPlan := plan.ToDBModel()
		// Look up each resource type and update it in the struct.
		for i, planQuotaDefault := range dbPlan.PlanQuotaDefaults {
			resourceType, err := db.GetResourceTypeByName(context, tx, planQuotaDefault.ResourceType.Name)
			if err != nil {
				return model.Error(ctx, err.Error(), http.StatusInternalServerError)
			}
			if resourceType == nil {
				msg := fmt.Sprintf("resource type not found: %s", resourceType.Name)
				return model.Error(ctx, msg, http.StatusBadRequest)
			}
			dbPlan.PlanQuotaDefaults[i].ResourceType = *resourceType

			log.Debugf("adding plan quota default resource %s to plan %s", resourceType.Name, plan.Name)
		}
		log.Debugf("translated plan: %+v", dbPlan)

		// Add the plan to the database.
		err := tx.WithContext(context).Create(&dbPlan).Error
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		log.Debug("successfully added plan to the database")

		return model.SuccessMessage(ctx, "Success", http.StatusOK)
	})
}

// AddPlanQuotaDefaults adds quota defaults to an exisitng subscription plan.
//
// swagger:route POST /plans/{plan_id}/quota-defaults plans addPlanQuotaDefaults
//
// # Add Plan Quota Default Values
//
// Adds quota default values to an existing plan. The existing quota default values for the plan will be left in
// place. The effective quota default value for a specific subscription plan for a specific resource type is always the
// quota default for that resource type with the most recent effective date not greater than the current date.
//
// Responses:
//
//	200: planResponse
//	400: badRequestResponse
//	404: notFoundResponse
//	500: internalServerErrorResponse
func (s Server) AddPlanQuotaDefaults(ctx echo.Context) error {
	var err error

	// Extract and validate the plan ID.
	planID, err := extractPlanID(ctx)
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusBadRequest)
	}

	// Initialize the logger and log a message indicating that the plan is being updated.
	log := log.WithFields(
		logrus.Fields{
			"context": "adding plan quota defaults",
			"plan_id": planID,
		},
	)
	log.Info("adding quota defaults to an existing plan")

	// Parse and validate the request body.
	var planQuotaDefaultList httpmodel.NewPlanQuotaDefaultList
	if err = ctx.Bind(&planQuotaDefaultList); err != nil {
		return model.Error(ctx, err.Error(), http.StatusBadRequest)

	}

	// Begin a transaction.
	return s.GORMDB.Transaction(func(tx *gorm.DB) error {
		context := ctx.Request().Context()

		// Verify that the plan exists.
		plan, err := db.GetPlanByID(context, tx, planID)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		} else if plan == nil {
			msg := fmt.Sprintf("plan ID %s not found", planID)
			return model.Error(ctx, msg, http.StatusNotFound)
		}

		// Convert the request body to the equivalent database model and insert the plan ID into each object.
		planQuotaDefaults := planQuotaDefaultList.ToDBModel()
		for _, pqd := range planQuotaDefaults {
			pqd.PlanID = &planID
		}

		// Retireve the list of resource types from the database.
		resourceTypeList, err := db.ListResourceTypes(context, tx)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		// Plug the actual resource types into each plan quota default.
		for _, pqd := range planQuotaDefaults {
			rt, err := resourceTypeList.GetResourceTypeByName(pqd.ResourceType.Name)
			if err != nil {
				return model.Error(ctx, err.Error(), http.StatusBadRequest)
			}
			pqd.ResourceType = *rt
		}

		// Save the list of resource types
		err = db.SavePlanQuotaDefaults(context, tx, planQuotaDefaults)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}

		// Look up the plan with the new plan quota defaults included and return it in the response.
		plan, err = db.GetPlanByID(context, tx, planID)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		} else if plan == nil {
			msg := fmt.Sprintf("plan ID %s not found after saving it", planID)
			return model.Error(ctx, msg, http.StatusInternalServerError)
		}
		return model.Success(ctx, plan, http.StatusOK)
	})
}
