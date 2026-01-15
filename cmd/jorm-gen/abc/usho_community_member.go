package mymodels

import (
	"time"
)

// UshoCommunityMember 代表数据库表 usho_community_member 的模型
type UshoCommunityMember struct {
	ID int64 `jorm:"column:id;pk;auto;notnull"`
	ComID int64 `jorm:"column:com_id;notnull"`
	Name string `jorm:"column:name;notnull"`
	UserID int64 `jorm:"column:user_id;notnull"`
	CountryCode string `jorm:"column:country_code;notnull"`
	ProvinceCode string `jorm:"column:province_code;notnull"`
	CityCode string `jorm:"column:city_code;notnull"`
	Title string `jorm:"column:title;notnull"`
	Integral int64 `jorm:"column:integral;notnull"`
	Forms string `jorm:"column:forms;notnull"`
	Intro string `jorm:"column:intro;notnull"`
	Remark string `jorm:"column:remark;notnull"`
	FromComID int64 `jorm:"column:from_com_id;notnull"`
	JoinedEvents string `jorm:"column:joined_events;notnull"`
	SelectAssistVerify string `jorm:"column:select_assist_verify;notnull"`
	VerifyMethod string `jorm:"column:verify_method;notnull"`
	ProfileCodes string `jorm:"column:profile_codes;notnull"`
	Failure string `jorm:"column:failure;notnull"`
	IsAdmin int64 `jorm:"column:is_admin;notnull"`
	IsChairman int64 `jorm:"column:is_chairman;notnull"`
	IsVerified int64 `jorm:"column:is_verified;notnull"`
	IsSubscribe int64 `jorm:"column:is_subscribe;notnull"`
	IsDeleted int64 `jorm:"column:is_deleted;notnull"`
	IsRejoin int64 `jorm:"column:is_rejoin;notnull"`
	VisitIp string `jorm:"column:visit_ip;notnull"`
	VisitAt time.Time `jorm:"column:visit_at"`
	EditAt time.Time `jorm:"column:edit_at"`
	VerifiedAt time.Time `jorm:"column:verified_at;notnull"`
	CreatedAt time.Time `jorm:"column:created_at;auto_time"`
	UpdateAt time.Time `jorm:"column:update_at"`
}
