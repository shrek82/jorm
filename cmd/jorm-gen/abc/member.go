package mymodels

import (
	"time"
)

// Member 代表数据库表 member 的模型
type Member struct {
	ID int64 `jorm:"column:id;pk;auto;notnull"`
	UserId string `jorm:"column:userId;notnull"`
	Name string `jorm:"column:name;notnull"`
	Mobile string `jorm:"column:mobile;notnull"`
	IdCard string `jorm:"column:idCard;notnull"`
	EnrolmentYear string `jorm:"column:enrolmentYear;notnull"`
	CollegeName string `jorm:"column:collegeName;notnull"`
	MajorName string `jorm:"column:majorName;notnull"`
	ClassName string `jorm:"column:className;notnull"`
	WorkUnit string `jorm:"column:workUnit;notnull"`
	UpdateAt time.Time `jorm:"column:updateAt;notnull"`
	CreateAt time.Time `jorm:"column:createAt;notnull"`
}
