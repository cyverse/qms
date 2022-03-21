package controllers

import (
	"fmt"
	"net/http"

	"gorm.io/gorm/clause"

	"github.com/cyverse-de/echo-middleware/v2/params"
	"github.com/cyverse/QMS/internal/db"
	"github.com/cyverse/QMS/internal/httpmodel"
	"github.com/cyverse/QMS/internal/model"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type PlanQuotaDefaultValues struct {
	PlanName         string  `json:"plan_name"`
	QuotaValue       float64 `json:"quota_value"`
	ResourceTypeName string  `json:"resource_type_name"`
}

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
// List Plans
//
// Lists all of the plans that are currently available.
//
// responses:
//   200: plansResponse
//   500: internalServerErrorResponse
func (s Server) GetAllPlans(ctx echo.Context) error {
	context := ctx.Request().Context()
	plans, err := db.ListPlans(context, s.GORMDB)
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}
	return model.Success(ctx, plans, http.StatusOK)
}

// GetPlanByID returns the plan with the given identifier.
//
// swagger:route GET /plans/{plan_id} plans getPlanByID
//
// Get Plan Information
//
// Returns the plan with the given identifier.
//
// responses:
//   200: planResponse
//   400: badRequestResponse
//   500: internalServerErrorResponse
func (s Server) GetPlanByID(ctx echo.Context) error {
	context := ctx.Request().Context()
	var err error
	// Extract and validate the plan ID.
	planID, err := extractPlanID(ctx)
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusBadRequest)
	}
	// Look up the plan.
	plan, err := db.GetPlanByID(context, s.GORMDB, planID)
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}
	if plan == nil {
		msg := fmt.Sprintf("plan ID %s not found", planID)
		return model.Error(ctx, msg, http.StatusNotFound)
	}

	return model.Success(ctx, plan, http.StatusOK)
}

// AddPlan adds a new plan to the database.
//
// swagger:route POST /v1/plans
func (s Server) AddPlan(ctx echo.Context) error {
	context := ctx.Request().Context()
	var err error
	// Parse and validate the request body.
	var plan httpmodel.NewPlan
	if err = ctx.Bind(&plan); err != nil {
		return model.Error(ctx, err.Error(), http.StatusBadRequest)
	}
	if err = plan.Validate(); err != nil {
		return model.Error(ctx, err.Error(), http.StatusBadRequest)
	}
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
		}
		// Add the plan to the database.
		err := tx.WithContext(context).Create(&dbPlan).Error
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}
		return model.SuccessMessage(ctx, "Success", http.StatusOK)
	})
}

func (s Server) AddPlanQuotaDefault(ctx echo.Context) error {
	var err error
	// Parse and validate the request body.
	var planQuotaDefaultValues PlanQuotaDefaultValues
	var context = ctx.Request().Context()
	if err = ctx.Bind(&planQuotaDefaultValues); err != nil {
		return model.Error(ctx, err.Error(), http.StatusBadRequest)
	}
	if planQuotaDefaultValues.PlanName == "" {
		return model.Error(ctx, "plan name can not be empty", http.StatusBadRequest)
	}
	if planQuotaDefaultValues.ResourceTypeName == "" {
		return model.Error(ctx, "resource type name can not be empty", http.StatusBadRequest)
	}

	return s.GORMDB.Transaction(func(tx *gorm.DB) error {
		plan, err := db.GetPlan(context, tx, planQuotaDefaultValues.PlanName)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}
		if plan == nil {
			msg := fmt.Sprintf("plan not found: %s", planQuotaDefaultValues.PlanName)
			return model.Error(ctx, msg, http.StatusBadRequest)
		}

		resourceType, err := db.GetResourceTypeByName(context, tx, planQuotaDefaultValues.ResourceTypeName)
		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}
		if resourceType == nil {
			msg := fmt.Sprintf("resource type not found: %s", planQuotaDefaultValues.ResourceTypeName)
			return model.Error(ctx, msg, http.StatusBadRequest)
		}
		planQuotaDefault := model.PlanQuotaDefault{
			PlanID:         plan.ID,
			ResourceTypeID: resourceType.ID,
			QuotaValue:     planQuotaDefaultValues.QuotaValue,
		}
		//updates quota value if quota value exists for a plan and resource type or else creates a defaults quota value for the plan and resource type.
		err = tx.WithContext(context).
			Debug().
			Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{
						Name: "plan_id",
					},
					{
						Name: "resource_type_id",
					},
				},
				DoUpdates: clause.AssignmentColumns([]string{"quota_value"}),
			}).
			Create(&planQuotaDefault).Error

		if err != nil {
			return model.Error(ctx, err.Error(), http.StatusInternalServerError)
		}
		return model.Success(ctx, "Success", http.StatusOK)
	})
}

type quotaReq struct {
	Username     string  `json:"user_name"`
	ResourceName string  `json:"resource_type_name"`
	QuotaValue   float64 `json:"quota_value"`
}

func (s Server) AddQuota(ctx echo.Context) error {
	context := ctx.Request().Context()

	var quotaReq quotaReq
	if err := ctx.Bind(&quotaReq); err != nil {
		return model.Error(ctx, err.Error(), http.StatusBadRequest)
	}

	if quotaReq.Username == "" {
		return model.Error(ctx, "invalid username", http.StatusBadRequest)
	}
	if quotaReq.ResourceName == "" {
		return model.Error(ctx, "invalid resource name", http.StatusBadRequest)
	}
	if quotaReq.QuotaValue < 0 {
		return model.Error(ctx, "invalid Quota value", http.StatusBadRequest)
	}
	resource, err := db.GetResourceTypeByName(context, s.GORMDB, quotaReq.ResourceName)
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}
	user, err := db.GetUser(context, s.GORMDB, quotaReq.Username)
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}
	userPlan, err := db.GetActiveUserPlan(context, s.GORMDB, user.Username)
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}
	var quota = model.Quota{
		UserPlanID:     userPlan.PlanID,
		Quota:          quotaReq.QuotaValue,
		ResourceTypeID: resource.ID,
	}
	err = s.GORMDB.WithContext(context).Debug().
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{
					Name: "user_plan_id",
				},
				{
					Name: "resource_type_id",
				},
			},
			DoUpdates: clause.AssignmentColumns([]string{"quota"}),
		}).
		Create(&quota).Error
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}
	return model.Success(ctx, "Success", http.StatusOK)
}
