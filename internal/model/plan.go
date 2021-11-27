package model

import (
	"time"

	"gorm.io/gorm"
)

type Users struct {
	gorm.Model
	id       string `gorm:"column:id;primaryKey;type:varchar(20)"`
	username string `gorm:"column:username;type:varchar(20)"`
}

func (u *Users) TableName() string {
	return "users"
}

type Plans struct {
	id   string    `gorm:"column:id;primaryKey;type:varchar(20)"`
	name string    `gorm:"column:name;type:varchar(20)"`
	date time.Time `gorm:"column:date;type:date"`
}

func (p *Plans) TableName() string {
	return "plan"
}

type Resource_types struct {
	id   string  `gorm:"column:id;primaryKey;type:varchar(20)"`
	name string  `gorm:"column:name;type:varchar(20)"`
	unit float64 `gorm:"column:unit;type:numeric"`
}

func (rt *Resource_types) TableName() string {
	return "resource_types"
}

type Plan_quota_defaults struct {
	id               string         `gorm:"column:id;primaryKey;type:varchar(20)"`
	plan_id          Plans          `gorm:"references:Plan"`
	resource_type_id Resource_types `gorm:"references:Resource_types"`
	quota_value      string         `gorm:"column:quota_value;type:varchar(20)"`
}

func (pqd *Plan_quota_defaults) TableName() string {
	return "plan_quota_defaults"
}

type User_plans struct {
	id                   string `gorm:"column:id;primaryKey;type:varchar(20)"`
	quota                int
	plan_id              Plans          `gorm:"references:Plan"`
	resource_type_id     Resource_types `gorm:"references:Resource_types"`
	effective_start_date time.Time      `gorm:"column:effective_start_date;type:date"`
	effective_end_date   time.Time      `gorm:"column:effective_start_date ;type:date"`
	added_by             string         `gorm:"column:added_by;type:varchar(20)"`
	record_date          time.Time      `gorm:"column:record_date;type:date"`
	last_modified        time.Time      `gorm:"column:last_modified;type:date"`
	last_modified_by     string         `gorm:"column:last_modified_by;type:varchar(20)"`
}

func (up *User_plans) TableName() string {
	return "user_plans"
}
