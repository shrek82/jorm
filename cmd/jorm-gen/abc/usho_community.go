package mymodels

import (
	"time"
)

// UshoCommunity 代表数据库表 usho_community 的模型
type UshoCommunity struct {
	ID int64 `jorm:"column:id;pk;auto;notnull"`
	Name string `jorm:"column:name;notnull"`
	TypeID int64 `jorm:"column:type_id;notnull"`
	LevelID int64 `jorm:"column:level_id;notnull"`
	OrgID string `jorm:"column:org_id;notnull"`
	OrgName string `jorm:"column:org_name;notnull"`
	Abbreviation string `jorm:"column:abbreviation;notnull"`
	ParentID int64 `jorm:"column:parent_id;notnull"`
	ParentIds string `jorm:"column:parent_ids;notnull"`
	SchoolID int64 `jorm:"column:school_id;notnull"`
	SchoolMotto string `jorm:"column:school_motto;notnull"`
	UserID int64 `jorm:"column:user_id;notnull"`
	PaymentType int64 `jorm:"column:payment_type;notnull"`
	PaymentStartDate time.Time `jorm:"column:payment_start_date"`
	PaymentEndDate time.Time `jorm:"column:payment_end_date"`
	FoundedDate time.Time `jorm:"column:founded_date;notnull"`
	CountryID int64 `jorm:"column:country_id;notnull"`
	ProvinceID int64 `jorm:"column:province_id;notnull"`
	CityID int64 `jorm:"column:city_id;notnull"`
	AreaID int64 `jorm:"column:area_id;notnull"`
	CoverPath string `jorm:"column:cover_path;notnull"`
	LogoPath string `jorm:"column:logo_path;notnull"`
	Fax string `jorm:"column:fax;notnull"`
	Tel string `jorm:"column:tel;notnull"`
	Email string `jorm:"column:email;notnull"`
	Address string `jorm:"column:address;notnull"`
	Contacts string `jorm:"column:contacts;notnull"`
	Website string `jorm:"column:website;notnull"`
	Weixin string `jorm:"column:weixin;notnull"`
	WeixinCodePath string `jorm:"column:weixin_code_path;notnull"`
	Weibo string `jorm:"column:weibo;notnull"`
	Principal string `jorm:"column:principal;notnull"`
	Intro string `jorm:"column:intro;notnull"`
	Keywords string `jorm:"column:keywords;notnull"`
	CreateReason string `jorm:"column:create_reason;notnull"`
	IsVerified int64 `jorm:"column:is_verified;notnull"`
	IsClosed int64 `jorm:"column:is_closed;notnull"`
	IsCertified int64 `jorm:"column:is_certified;notnull"`
	IsPfixed int64 `jorm:"column:is_pfixed;notnull"`
	IsDeleted int64 `jorm:"column:is_deleted;notnull"`
	CollectionAt time.Time `jorm:"column:collection_at"`
	CreatedAt time.Time `jorm:"column:created_at;auto_time"`
	UpdateAt time.Time `jorm:"column:update_at"`
	ContactsCode string `jorm:"column:contacts_code;notnull"`
}
