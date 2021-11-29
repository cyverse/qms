package model

type Users struct {
	ID       string `gorm:"column:id;primaryKey;type:uuid"`
	UserName string `gorm:"column:username;type:varchar(100)"`
}

func (u *Users) TableName() string {
	return "users"
}
