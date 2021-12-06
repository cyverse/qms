package controllers

import (
	"net/http"
	"strings"
	"time"

	"github.com/cyverse/QMS/internal/model"
	"github.com/labstack/echo"
)

type AdminQuotaDetails struct {
	UserName     string
	UserID       string
	PlanName     string
	Quota        float64
	ResourceName string
	Unit         string
}

type AdminUsageDetails struct {
	UserName     string
	UserID       string
	PlanName     string
	ResourceName string
	Usage        float64
	Unit         string
}

type UpdateUsagesReq struct {
	UserName             string  `json:"username"`
	ResourceType         string  `json:"resource_type"`
	UpdateType           string  `json:"update_type"`
	UsageAdjustmentValue float64 `json:"usgae_adjustment_value"`
	EffectiveDate        string  `json:"effective_date"`
}

type UpdateQuotaReq struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
}

func (s Server) UpdateQuota(ctx echo.Context) error {
	quotaid := ctx.Param("quotaid")
	if quotaid == "" {
		return ctx.JSON(http.StatusBadRequest, model.ErrorResponse("Invalid QuotaID", http.StatusBadRequest))
	}

	req := UpdateQuotaReq{}
	err := ctx.Bind(&req)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, model.ErrorResponse(err.Error(), http.StatusBadRequest))
	}

	quota := model.Quotas{}

	err = s.GORMDB.Debug().Where("id = ?", quotaid).Find(&quota).Error
	if err != nil {
		if strings.Contains(err.Error(), "invalid input") {
			return ctx.JSON(http.StatusBadRequest, model.ErrorResponse(err.Error(), http.StatusBadRequest))
		}
		return ctx.JSON(http.StatusInternalServerError, model.ErrorResponse(err.Error(), http.StatusInternalServerError))
	}

	if quota.ID != quotaid {
		return ctx.JSON(http.StatusBadRequest, model.ErrorResponse("Quota Not Found", http.StatusBadRequest))
	}

	value := req.Value
	if req.Type == "sub" {
		value = -1 * value
	}

	quota.Quota += value

	err = s.GORMDB.Debug().Updates(&quota).Error
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, model.ErrorResponse(err.Error(), http.StatusInternalServerError))
	}

	return ctx.JSON(http.StatusOK, model.SuccessResponse("Success", http.StatusOK))

}

func (s Server) UpdateUsages(ctx echo.Context) error {
	req := UpdateUsagesReq{}
	err := ctx.Bind(&req)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, model.ErrorResponse(err.Error(), http.StatusBadRequest))
	}
	effectivedate, err := time.Parse("2006-01-02", req.EffectiveDate)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, model.ErrorResponse(err.Error(), http.StatusBadRequest))
	}

	now := time.Now().Format("2006-01-02")

	usageDetails := []model.Usages{}

	err = s.GORMDB.Debug().Raw(`select usages.* from user_plans 		
		join usages on user_plans.id=usages.user_plan_id 
		join quotas on user_plans.id=quotas.user_plan_id 
		join resource_types on resource_types.id=quotas.resource_type_id
		join users on users.id = user_plans.user_id
		where 
		user_plans.effective_start_date <=? and 
		user_plans.effective_end_date >=? and
		users.username = ? and 
		resource_types.name = ?`, now+"T00:00:00", now+"T23:59:59", req.UserName, req.ResourceType).Scan(&usageDetails).Error
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, model.ErrorResponse(err.Error(), http.StatusInternalServerError))
	}

	for _, usagerec := range usageDetails {
		usagerec.UpdatedAt = effectivedate
		value := req.UsageAdjustmentValue
		if req.UpdateType == "sub" {
			value = -1 * value
		}
		usagerec.Usage += value
		err := s.GORMDB.Debug().Updates(&usagerec).Error
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, model.ErrorResponse(err.Error(), http.StatusInternalServerError))
		}
	}

	return ctx.JSON(http.StatusOK, model.SuccessResponse("Success", http.StatusOK))

}

func (s Server) GetAllActiveQuotas(ctx echo.Context) error {
	resource := ctx.QueryParam("resource")
	resourcefilter := ""
	if resource != "" {
		resourcefilter = ` and resource_types.name = '` + resource + `'`
	}
	username := ctx.QueryParam("username")
	usernamefilter := ""
	if username != "" {
		usernamefilter = ` and users.username = '` + username + `'`
	}
	now := time.Now().Format("2006-01-02")

	plandata := []AdminQuotaDetails{}

	err := s.GORMDB.Debug().Raw(`select users.username as user_name, users.id as user_id, plans.name as plan_name, quotas.quota, resource_types.name as resource_name, resource_types.unit from user_plans
	join plans on plans.id = user_plans.plan_id	
	join quotas on user_plans.id=quotas.user_plan_id
	join resource_types on resource_types.id=quotas.resource_type_id
	join users on users.id = user_plans.user_id
	where
	user_plans.effective_start_date <=? and
	user_plans.effective_end_date >=? `+usernamefilter+resourcefilter, now+"T00:00:00", now+"T23:59:59").Scan(&plandata).Error
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, model.ErrorResponse(err.Error(), http.StatusInternalServerError))
	}
	return ctx.JSON(http.StatusOK, model.SuccessResponse(plandata, http.StatusOK))
}

func (s Server) GetAllActiveUsage(ctx echo.Context) error {
	resource := ctx.QueryParam("resource")
	resourcefilter := ""
	if resource != "" {
		resourcefilter = ` and resource_types.name = '` + resource + `'`
	}
	username := ctx.QueryParam("username")
	usernamefilter := ""
	if username != "" {
		usernamefilter = ` and users.username = '` + username + `'`
	}
	now := time.Now().Format("2006-01-02")

	plandata := []AdminUsageDetails{}

	err := s.GORMDB.Debug().Raw(`select users.username as user_name, users.id as user_id, plans.name as plan_name, usages.usage, resource_types.unit, resource_types.name as resource_name from user_plans 
		join plans on plans.id = user_plans.plan_id 
		join usages on user_plans.id=usages.user_plan_id 
		join quotas on user_plans.id=quotas.user_plan_id 
		join resource_types on resource_types.id=quotas.resource_type_id
		join users on users.id = user_plans.user_id
		where 
		user_plans.effective_start_date <=? and 
		user_plans.effective_end_date >=? `+usernamefilter+resourcefilter, now+"T00:00:00", now+"T23:59:59").Scan(&plandata).Error
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, model.ErrorResponse(err.Error(), http.StatusInternalServerError))
	}
	return ctx.JSON(http.StatusOK, model.SuccessResponse(plandata, http.StatusOK))

}
