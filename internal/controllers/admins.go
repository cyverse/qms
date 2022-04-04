package controllers

import (
	"github.com/cyverse/QMS/internal/db"
	"net/http"

	"github.com/cyverse/QMS/internal/model"
	"github.com/labstack/echo/v4"
)

func (s Server) GetAllUsageOfUser(ctx echo.Context) error {
	var err error
	username := ctx.Param("username")
	if username == "" {
		return model.Error(ctx, "invalid username", http.StatusBadRequest)
	}
	user, err := db.GetUser(s.GORMDB, username)
	activePlan, err := db.GetActiveUserPlan(s.GORMDB, username)
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}
	var userPlan model.UserPlan
	err = s.GORMDB.
		Preload("Usages").
		Preload("Usages.ResourceType").
		Where("user_id=?", user.ID).
		Where("plan_id=?", activePlan.PlanID).
		Find(&userPlan).Error

	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}
	return model.Success(ctx, userPlan.Usages, http.StatusOK)
}

func (s Server) GetAllActiveUserPlans(ctx echo.Context) error {
	var userPlans []model.UserPlan
	err := s.GORMDB.
		Preload("User").
		Preload("Plan").
		Preload("Plan.PlanQuotaDefaults").
		Preload("Plan.PlanQuotaDefaults.ResourceType").
		Preload("Quotas").
		Preload("Quotas.ResourceType").
		Preload("Usages").
		Preload("Usages.ResourceType").
		Where(
			s.GORMDB.Where("CURRENT_TIMESTAMP BETWEEN user_plans.effective_start_date AND user_plans.effective_end_date").
				Or("CURRENT_TIMESTAMP > user_plans.effective_start_date AND user_plans.effective_end_date IS NULL")).
		Find(&userPlans).Error
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}
	return model.Success(ctx, userPlans, http.StatusOK)
}
