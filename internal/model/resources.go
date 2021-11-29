package model

type ResourceTypes struct {
	ID   string `gorm:"column:id;primaryKey;type:uuid"`
	Name string `gorm:"column:name;type:varchar(100)"`
	Unit string `gorm:"column:unit;type:varchar(100)"`
}

func (rt *ResourceTypes) TableName() string {
	return "resource_types"
}
