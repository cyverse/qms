package controllers

import (
	"net/http"

	"github.com/cyverse/QMS/internal/db"

	"github.com/cyverse/QMS/internal/model"
	"github.com/labstack/echo/v4"
)

func (s Server) GetAllUsageOfUser(ctx echo.Context) error {
	var err error
	context := ctx.Request().Context()
	username := ctx.Param("username")
	if username == "" {
		return model.Error(ctx, "invalid username", http.StatusBadRequest)
	}

	var user model.User
	err = s.GORMDB.WithContext(context).Where("username=?", username).Find(&user).Error
	if err != nil {
		return model.Error(ctx, "user name not found", http.StatusInternalServerError)
	}
	activePlan, err := db.GetActiveUserPlan(context, s.GORMDB, username)
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}
	var userPlan model.UserPlan
	err = s.GORMDB.WithContext(context).
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
	context := ctx.Request().Context()
	var userPlans []model.UserPlan
	err := s.GORMDB.WithContext(context).
		Preload("User").
		Preload("Plan").
		Preload("Plan.PlanQuotaDefaults").
		Preload("Plan.PlanQuotaDefaults.ResourceType").
		Preload("Quotas").
		Preload("Quotas.ResourceType").
		Preload("Usages").
		Preload("Usages.ResourceType").
		Where(
			s.GORMDB.WithContext(context).
				Where("CURRENT_TIMESTAMP BETWEEN user_plans.effective_start_date AND user_plans.effective_end_date").
				Or("CURRENT_TIMESTAMP > user_plans.effective_start_date AND user_plans.effective_end_date IS NULL")).
		Find(&userPlans).Error
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}
	return model.Success(ctx, userPlans, http.StatusOK)
}
