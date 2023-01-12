package controllers

import (
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/cyverse/QMS/internal/model"
	"github.com/labstack/echo/v4"
)

func (s Server) GetAllActiveSubscriptions(ctx echo.Context) error {
	log := log.WithFields(logrus.Fields{"context": "getting all active user plans"})

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
				Where("CURRENT_TIMESTAMP BETWEEN subscriptions.effective_start_date AND subscriptions.effective_end_date").
				Or("CURRENT_TIMESTAMP > subscriptions.effective_start_date AND subscriptions.effective_end_date IS NULL")).
		Find(&userPlans).Error
	if err != nil {
		return model.Error(ctx, err.Error(), http.StatusInternalServerError)
	}

	log.Debug("got user plans from the database")

	return model.Success(ctx, userPlans, http.StatusOK)
}
