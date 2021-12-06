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
	return ctx.JSON(http.StatusOK, model.SuccessResponse(data, http.StatusOK))
}

type PlanDetails struct {
	Name  string
	Usage string
	Quota float64
	Unit  string
}
type QuotaDetails struct {
	PlanName     string
	Quota        float64
	ResourceName string
	Unit         string
}

type UsageDetails struct {
	PlanName     string
	ResourceName string
	Usage        string
	Unit         string
}

func (s Server) GetUserPlanDetails(ctx echo.Context) error {
	username := ctx.Param("username")
	if username == "" {
		return ctx.JSON(http.StatusBadRequest, model.ErrorResponse("Invalid Usernaem", http.StatusBadRequest))
	}
	now := time.Now().Format("2006-01-02")

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
		return ctx.JSON(http.StatusInternalServerError, model.ErrorResponse(err.Error(), http.StatusInternalServerError))
	}
	return ctx.JSON(http.StatusOK, model.SuccessResponse(plandata, http.StatusOK))
}
func (s Server) GetUserAllQuotas(ctx echo.Context) error {
	resource := ctx.QueryParam("resource")
	resourceFilter := ""
	if resource != "" {
		resourceFilter = `and resource_types.name=?`
	}
	username := ctx.Param("username")
	if username == "" {
		return ctx.JSON(http.StatusBadRequest, model.ErrorResponse("Invalid Usernaem", http.StatusBadRequest))
	}

	plandata := []QuotaDetails{}
	err := s.GORMDB.Raw(`select plans.name as plan_name,quotas.quota,resource_types.name as resource_nam,resource_types.unit as resource_unit from
	user_plans
	join plans on plans.id=user_plans.plan_id
	join quotas on user_plans.id=quotas.user_plan_id
	join resource_types on resource_types.id=quotas.resource_type_id
	join users on users.id=user_plans.user_id
	where
	users.username=?`+resourceFilter, username).Scan(&plandata).Error
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}
	return ctx.JSON(http.StatusOK, model.SuccessResponse(plandata, http.StatusOK))
}
func (s Server) GetUserQuotaDetails(ctx echo.Context) error {
	quotaid := ctx.Param("quotaid")
	if quotaid == "" {
		return ctx.JSON(http.StatusBadRequest, model.ErrorResponse("Invalid quota Id", http.StatusBadRequest))
	}

	username := ctx.Param("username")
	if username == "" {
		return ctx.JSON(http.StatusBadRequest, model.ErrorResponse("Invalid Usrname", http.StatusBadRequest))
	}

	plandata := []PlanDetails{}
	err := s.GORMDB.Raw(`select plans.name as plan_name,quotas.quota,resource_types.name as resource_nam,resource_types.unit as resource_unit from
	user_plans
	join plans on plans.id=user_plans.plan_id
	join quotas on user_plans.id=quotas.user_plan_id
	join resource_types on resource_types.id=quotas.resource_type_id
	join users on users.id=user_plans.user_id
	where
	users.username=? and quotas.id=?`, username, quotaid).Scan(&plandata).Error
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}
	return ctx.JSON(http.StatusOK, model.SuccessResponse(plandata, http.StatusOK))
}
func (s Server) GetUserUsageDetails(ctx echo.Context) error {
	resource := ctx.QueryParam("resource")
	resourceFilter := ""
	if resource != "" {
		resourceFilter = `and resource_types.name=?`
	}
	username := ctx.Param("username")
	if username == "" {
		return ctx.JSON(http.StatusBadRequest, model.ErrorResponse("Invalid Usrname", http.StatusBadRequest))
	}

	plandata := []UsageDetails{}
	err := s.GORMDB.Raw(`select plans.name as plan_name,usages.usageu,resource_types.name as resource_nam,resource_types.unit as resource_unit from
	user_plans
	join plans on plans.id=user_plans.plan_id
	join usages on user_plans.id=usages.user_plan_id
	join quotas on user_plans.id=quotas.user_plan_id
	join resource_types on resource_types.id=quotas.resource_type_id
	join users on users.id=user_plans.user_id
	where
	users.username=?`+resourceFilter, username, resource).Scan(&plandata).Error
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}
	return ctx.JSON(http.StatusOK, model.SuccessResponse(plandata, http.StatusOK))
}
