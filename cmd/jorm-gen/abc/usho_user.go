package mymodels

import (
	"time"
)

// UshoUser 代表数据库表 usho_user 的模型
type UshoUser struct {
	ID int64 `jorm:"column:id;pk;auto;notnull"`
	Sex int64 `jorm:"column:sex;notnull"`
	Name string `jorm:"column:name;notnull"`
	Nickname string `jorm:"column:nickname;notnull"`
	Password string `jorm:"column:password;notnull"`
	Birthday string `jorm:"column:birthday;notnull"`
	AvatarPath string `jorm:"column:avatar_path;notnull"`
	Source string `jorm:"column:source;notnull"`
	IsVirtual int64 `jorm:"column:is_virtual;notnull"`
	IsClosed int64 `jorm:"column:is_closed;notnull"`
	IsDeleted int64 `jorm:"column:is_deleted;notnull"`
	LastVisitAt time.Time `jorm:"column:last_visit_at;notnull"`
	LastLoginAt time.Time `jorm:"column:last_login_at;notnull"`
	LastLoginIp string `jorm:"column:last_login_ip;notnull"`
	CreatedAt time.Time `jorm:"column:created_at;auto_time"`
	UpdateAt time.Time `jorm:"column:update_at"`
	ProvinceID int64 `jorm:"column:province_id;notnull"`
	CityID int64 `jorm:"column:city_id;notnull"`
	DistrictID int64 `jorm:"column:district_id;notnull"`
	Hobby string `jorm:"column:hobby;notnull"`
	Email string `jorm:"column:email;notnull"`
	Weixin string `jorm:"column:weixin;notnull"`
	Address string `jorm:"column:address;notnull"`
	Education int64 `jorm:"column:education;notnull"`
	StudentCode string `jorm:"column:student_code;notnull"`
	EnrollmentYear string `jorm:"column:enrollment_year;notnull"`
	GraduationYear string `jorm:"column:graduation_year;notnull"`
	Organization string `jorm:"column:organization;notnull"`
	Speciality string `jorm:"column:speciality;notnull"`
	Class string `jorm:"column:class;notnull"`
	CertificateImg string `jorm:"column:certificate_img;notnull"`
	Unit string `jorm:"column:unit;notnull"`
	Position string `jorm:"column:position;notnull"`
	Department string `jorm:"column:department;notnull"`
	UnitNatureID int64 `jorm:"column:unit_nature_id;notnull"`
	UnitSize int64 `jorm:"column:unit_size;notnull"`
	UnitIndustryID int64 `jorm:"column:unit_industry_id;notnull"`
	UnitAddress string `jorm:"column:unit_address;notnull"`
	PersonalSignature string `jorm:"column:personal_signature;notnull"`
}
