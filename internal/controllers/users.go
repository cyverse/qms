package controllers

import (
	"net/http"
	"time"

	"github.com/cyverse/QMS/internal/model"
	"github.com/labstack/echo"
)

func (s Server) GetAllUsers(ctx echo.Context) error {
	data := []model.Users{}
	err := s.GORMDB.Debug().Find(&data).Error
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}
	return ctx.JSON(http.StatusOK, data)
}

func (s Server) GetUserPlanDetails(ctx echo.Context) error {
	username := ctx.Param("username")
	now := time.Now().Format("2006-01-02")
	type PlanDetails struct {
		Name  string
		Usage string
		Quota float64
		Unit  string
	}
	plandata := []PlanDetails{}
	err := s.GORMDB.Raw(`select plans.name,usage.usage,quotas.quota,resource_types.unit from
	user_plans
	join plans on plans.id=user_plans.plan_id
	join usages on user_plans.id=usages.user_plan_id
	join quotas on user_plans.id=usages.user_plan_id
	join resource_types on resource_types.id=quotas.resource_type_id
	join users on users.id=user_plans.user_id
	where
	user_plans.effective_start_date<=? and
	user_plans.effective_end_date>=? and
	users.username=?`, now+"T00:00:00", now+"T23:59:59", username).Scan(&plandata).Error
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}
	return ctx.JSON(http.StatusOK, plandata)
}
