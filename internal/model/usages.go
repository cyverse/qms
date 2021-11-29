package model

import (
	"gorm.io/gorm"
)

type Usages struct {
	gorm.Model
	ID             string        `gorm:"column:id;primaryKey;type:uuid"`
	Usage          float64       `gorm:"column:usage;type:numeric"`
	AddedBy        string        `gorm:"column:added_by;type:varchar(100)"`
	LastModifiedBy string        `gorm:"column:last_modified_by;type:varchar(100)"`
	UserPlanID     string        `gorm:"unique"`
	UserPlans      UserPlans     `gorm:"foreignKey:UserPlanID;references:ID;"`
	ResourceTypeID string        `gorm:"unique"`
	ResourceTypes  ResourceTypes `gorm:"foreignKey:ResourceTypeID;references:ID"`
}

func (u *Usages) TableName() string {
	return "usages"
}
