package mymodels

import (
	"time"
)

// UshoActivityType 代表数据库表 usho_activity_type 的模型
type UshoActivityType struct {
	ID int64 `jorm:"column:id;pk;auto;notnull"`
	Name string `jorm:"column:name;notnull"`
	IconPath string `jorm:"column:icon_path;notnull"`
	Intro string `jorm:"column:intro;notnull"`
	UserID int64 `jorm:"column:user_id;notnull"`
	OrderNum int64 `jorm:"column:order_num;notnull"`
	IsDeleted int64 `jorm:"column:is_deleted;notnull"`
	CreatedAt time.Time `jorm:"column:created_at;auto_time"`
	UpdateAt time.Time `jorm:"column:update_at"`
}
