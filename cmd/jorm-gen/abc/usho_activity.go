package mymodels

import (
	"time"
)

// UshoActivity 代表数据库表 usho_activity 的模型
type UshoActivity struct {
	ID int64 `jorm:"column:id;pk;auto;notnull"`
	Code string `jorm:"column:code;notnull"`
	ComID int64 `jorm:"column:com_id;notnull"`
	GroupID int64 `jorm:"column:group_id;notnull"`
	UserID int64 `jorm:"column:user_id;notnull"`
	TypeID int64 `jorm:"column:type_id;notnull"`
	Title string `jorm:"column:title;notnull"`
	StartDate time.Time `jorm:"column:start_date;notnull"`
	EndDate time.Time `jorm:"column:end_date;notnull"`
	SignupStartDate time.Time `jorm:"column:signup_start_date;notnull"`
	SignupEndDate time.Time `jorm:"column:signup_end_date;notnull"`
	RefundEndDate time.Time `jorm:"column:refund_end_date"`
	RefundMethod string `jorm:"column:refund_method;notnull"`
	OrganizerType string `jorm:"column:organizer_type;notnull"`
	Fee string `jorm:"column:fee;notnull"`
	PaymentMethod int64 `jorm:"column:payment_method;notnull"`
	Places int64 `jorm:"column:places;notnull"`
	LastSequence int64 `jorm:"column:last_sequence;notnull"`
	MaxNumber int64 `jorm:"column:max_number;notnull"`
	SignRange int64 `jorm:"column:sign_range;notnull"`
	Address string `jorm:"column:address;notnull"`
	AddressDetail string `jorm:"column:address_detail;notnull"`
	LocationLng string `jorm:"column:location_lng;notnull"`
	LocationLat string `jorm:"column:location_lat;notnull"`
	Map string `jorm:"column:map;notnull"`
	Intro string `jorm:"column:intro;notnull"`
	ContentType string `jorm:"column:content_type;notnull"`
	TagsIds string `jorm:"column:tags_ids;notnull"`
	FormIds string `jorm:"column:form_ids;notnull"`
	Hits int64 `jorm:"column:hits;notnull"`
	Heat int64 `jorm:"column:heat;notnull"`
	Heat2 int64 `jorm:"column:heat2;notnull"`
	IcoPath string `jorm:"column:ico_path;notnull"`
	ImgPath string `jorm:"column:img_path;notnull"`
	QrcodePath string `jorm:"column:qrcode_path;notnull"`
	Settings string `jorm:"column:settings;notnull"`
	UvNum int64 `jorm:"column:uv_num;notnull"`
	SignNum int64 `jorm:"column:sign_num;notnull"`
	SigninNum int64 `jorm:"column:signin_num;notnull"`
	InterestNum int64 `jorm:"column:interest_num;notnull"`
	CommentNum int64 `jorm:"column:comment_num;notnull"`
	PhotoNum int64 `jorm:"column:photo_num;notnull"`
	CollectNum int64 `jorm:"column:collect_num;notnull"`
	TeamsNum int64 `jorm:"column:teams_num;notnull"`
	ShareNum int64 `jorm:"column:share_num;notnull"`
	GoodNum int64 `jorm:"column:good_num;notnull"`
	NoticeQuota int64 `jorm:"column:notice_quota;notnull"`
	Qualification string `jorm:"column:qualification;notnull"`
	IsPublic int64 `jorm:"column:is_public;notnull"`
	IsVerified int64 `jorm:"column:is_verified;notnull"`
	IsSignVerify int64 `jorm:"column:is_sign_verify;notnull"`
	IsMultPoint int64 `jorm:"column:is_mult_point;notnull"`
	IsSuspend int64 `jorm:"column:is_suspend;notnull"`
	IsSuspendSignup int64 `jorm:"column:is_suspend_signup;notnull"`
	IsCmd int64 `jorm:"column:is_cmd;notnull"`
	IsFixed int64 `jorm:"column:is_fixed;notnull"`
	IsPaid int64 `jorm:"column:is_paid;notnull"`
	IsOpenTeam int64 `jorm:"column:is_open_team;notnull"`
	IsCanCreateTeam int64 `jorm:"column:is_can_create_team;notnull"`
	EnableAnonymous int64 `jorm:"column:enable_anonymous;notnull"`
	Credits int64 `jorm:"column:credits;notnull"`
	RewardfulAmount float64 `jorm:"column:rewardful_amount;notnull"`
	IsDraft int64 `jorm:"column:is_draft;notnull"`
	IsCredits int64 `jorm:"column:is_credits;notnull"`
	IsRewardful int64 `jorm:"column:is_rewardful;notnull"`
	IsGoodRecmd int64 `jorm:"column:is_good_recmd;notnull"`
	IsProjectRecmd int64 `jorm:"column:is_project_recmd;notnull"`
	IsSkipChargeMember int64 `jorm:"column:is_skip_charge_member;notnull"`
	IsClosed int64 `jorm:"column:is_closed;notnull"`
	IsDeleted int64 `jorm:"column:is_deleted;notnull"`
	RewardfulMsg string `jorm:"column:rewardful_msg;notnull"`
	VerifiedAt time.Time `jorm:"column:verified_at;notnull"`
	CreatedAt time.Time `jorm:"column:created_at;auto_time"`
	UpdateAt time.Time `jorm:"column:update_at"`
	ShowSignMember int64 `jorm:"column:show_sign_member;notnull"`
	IsOpenGroupChat int64 `jorm:"column:is_open_group_chat;notnull"`
}
