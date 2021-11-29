package model

import (
	"time"

	"gorm.io/gorm"
)

type Plans struct {
	ID          string `gorm:"column:id;primaryKey,unique;type:uuid"`
	Name        string `gorm:"column:name;type:varchar(100)"`
	Description string `gorm:"column:description;type:text"`
}

func (p *Plans) TableName() string {
	return "plans"
}

type PlanQuotaDefaults struct {
	gorm.Model
	ID             string  `gorm:"column:id;primaryKey;type:uuid"`
	QuotaValue     float64 `gorm:"column:quota_value;type:numeric"`
	PlanID         string
	Plans          Plans `gorm:"foreignKey:PlanID;references:ID;"`
	ResourceTypeID string
	ResourceTypes  ResourceTypes `gorm:"foreignKey:ResourceTypeID;references:ID"`
}

func (pqd *PlanQuotaDefaults) TableName() string {
	return "plan_quota_defaults"
}

type UserPlans struct {
	gorm.Model
	ID                 string    `gorm:"column:id;primaryKey;type:uuid"`
	EffectiveStartDate time.Time `gorm:"column:effective_start_date;type:date"`
	EffectiveEndDate   time.Time `gorm:"column:effective_end_date;type:date"`
	AddedBy            string    `gorm:"column:added_by;type:varchar(100)"`
	LastModifiedBy     string    `gorm:"column:last_modified_by;type:varchar(100)"`
	UserID             string
	Users              Users `gorm:"foreignKey:UserID;references:ID;"`
	PlanID             string
	Plans              Plans `gorm:"foreignKey:PlanID;references:ID;"`
}

func (up *UserPlans) TableName() string {
	return "user_plans"
}
