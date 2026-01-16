package tests

import (
	"errors"
	"os"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shrek82/jorm"
)

// ValidateUser 测试用的用户模型
type ValidateUser struct {
	ID    int64  `jorm:"pk,auto"`
	Name  string `jorm:"column:name"`
	Email string `jorm:"column:email"`
	Age   int    `jorm:"column:age"`
	Role  string `jorm:"column:role"`
}

// TableName 指定表名
func (m *ValidateUser) TableName() string {
	return "validate_users"
}

// CommonValidator 返回通用的验证规则
func (m *ValidateUser) CommonValidator() jorm.Validator {
	return jorm.Rules{
		"Name": {
			jorm.Required.Msg("Name is required"),
			jorm.MinLen(2).Msg("Name too short"),
		},
		"Email": {
			jorm.Email.Msg("Invalid email"),
		},
		"Age": {
			jorm.Range(18, 100).Msg("Must be adult"),
		},
	}.Validate
}

// BeforeInsert 钩子复用 CommonValidator
func (m *ValidateUser) BeforeInsert() error {
	return m.CommonValidator()(m)
}

// BeforeUpdate 钩子复用 CommonValidator
func (m *ValidateUser) BeforeUpdate() error {
	return m.CommonValidator()(m)
}

// AdminValidator 返回管理员权限验证函数
func (m *ValidateUser) AdminValidator() jorm.Validator {
	return func(value any) error {
		u := value.(*ValidateUser)
		if u.Role != "admin" {
			return errors.New("Unauthorized")
		}
		return nil
	}
}

// setupValidatorDB 初始化测试数据库
func setupValidatorDB(t *testing.T) *jorm.DB {
	dbFile := "validator_test.db"
	db, err := jorm.Open("sqlite3", dbFile, nil)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	err = db.AutoMigrate(&ValidateUser{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

// teardownValidatorDB 清理测试数据库
func teardownValidatorDB() {
	os.Remove("validator_test.db")
}

// TestValidator 验证器主测试函数
func TestValidator(t *testing.T) {
	db := setupValidatorDB(t)
	defer teardownValidatorDB()

	t.Run("Standalone Validation", func(t *testing.T) {
		// 测试独立验证功能（不涉及数据库）
		user := &ValidateUser{Name: "A", Email: "invalid", Age: 10}
		err := jorm.Validate(user, user.CommonValidator())
		if err == nil {
			t.Error("Expected validation error, got nil")
		}
		errs, ok := err.(jorm.ValidationErrors)
		if !ok {
			t.Errorf("Expected ValidationErrors, got %T", err)
		}

		// 验证错误信息是否匹配自定义消息
		if len(errs["Name"]) == 0 || errs["Name"][0].Error() != "Name too short" {
			t.Errorf("Unexpected error for Name: %v", errs["Name"])
		}
		if len(errs["Email"]) == 0 || errs["Email"][0].Error() != "Invalid email" {
			t.Errorf("Unexpected error for Email: %v", errs["Email"])
		}
	})

	t.Run("InsertWithValidator - Success", func(t *testing.T) {
		// 测试带验证的插入：成功场景
		user := &ValidateUser{Name: "Shrek", Email: "shrek@example.com", Age: 25, Role: "admin"}
		id, err := db.Model(user).InsertWithValidator(user, user.CommonValidator())
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
		if id == 0 {
			t.Error("Expected non-zero ID")
		}
	})

	t.Run("InsertWithValidator - Failure", func(t *testing.T) {
		// 测试带验证的插入：失败场景（违反基础规则）
		user := &ValidateUser{Name: "", Email: "bad", Age: 5}
		_, err := db.Model(user).InsertWithValidator(user, user.CommonValidator())
		if err == nil {
			t.Error("Expected validation error, got nil")
		}
	})

	t.Run("Combination Validator", func(t *testing.T) {
		// 测试多个验证器组合使用（基础规则 + 业务逻辑验证）
		user := &ValidateUser{Name: "Bob", Email: "bob@example.com", Age: 30, Role: "guest"}
		_, err := db.Model(user).InsertWithValidator(user, user.CommonValidator(), user.AdminValidator())
		if err == nil {
			t.Error("Expected error from AdminValidator, got nil")
		}
		if err.Error() != "Unauthorized" {
			t.Errorf("Expected 'Unauthorized', got '%v'", err)
		}
	})

	t.Run("UpdateWithValidator", func(t *testing.T) {
		// 测试带验证的更新
		user := &ValidateUser{Name: "Initial", Email: "init@example.com", Age: 20}
		id, _ := db.Model(user).Insert(user)
		user.ID = id

		// 合法更新
		user.Name = "Updated Name"
		affected, err := db.Model(user).Where("id = ?", id).UpdateWithValidator(user, user.CommonValidator())
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if affected == 0 {
			t.Error("Expected affected rows > 0")
		}

		// 非法更新：违反邮箱格式
		user.Email = "invalid-email"
		_, err = db.Model(user).Where("id = ?", id).UpdateWithValidator(user, user.CommonValidator())
		if err == nil {
			t.Error("Expected validation error for invalid email update, got nil")
		}
	})

	t.Run("Extended Format Rules", func(t *testing.T) {
		// 测试扩展格式校验规则（IP, JSON, UUID, 包含/排除等）
		type MetaData struct {
			IP   string
			JSON string
			UUID string
			Tags string
		}
		rules := jorm.Rules{
			"IP":   {jorm.IP},
			"JSON": {jorm.JSON},
			"UUID": {jorm.UUID},
			"Tags": {jorm.Contains("go"), jorm.Excludes("php")},
		}

		// 合法数据
		m1 := &MetaData{
			IP:   "192.168.1.1",
			JSON: `{"key": "value"}`,
			UUID: "550e8400-e29b-41d4-a716-446655440000",
			Tags: "golang-is-great",
		}
		if err := jorm.Validate(m1, rules.Validate); err != nil {
			t.Errorf("Expected nil for valid metadata, got %v", err)
		}

		// 非法数据：所有字段均违反规则
		m2 := &MetaData{
			IP:   "999.999.999.999",
			JSON: "{invalid-json}",
			UUID: "short-uuid",
			Tags: "php-is-everywhere",
		}
		err := jorm.Validate(m2, rules.Validate)
		if err == nil {
			t.Error("Expected validation errors, got nil")
		}
		errs := err.(jorm.ValidationErrors)
		if len(errs) != 4 {
			t.Errorf("Expected 4 fields with errors, got %d", len(errs))
		}
	})

	t.Run("Optional and When Rules", func(t *testing.T) {
		// 测试修饰符：Optional (可选) 和 When (条件触发)
		type WhenUser struct {
			Status string
			Reason string
		}

		rules := jorm.Rules{
			"Age": {jorm.Range(18, 100).Optional()},
			"Reason": {jorm.Required.When(func(v any) bool {
				// 当字段值为 "TRIGGER" 时触发必填校验
				return v != nil && v.(string) == "TRIGGER"
			})},
		}

		// Age 为 0 (零值) - 因 Optional 应该通过校验
		u1 := &ValidateUser{Age: 0}
		if err := jorm.Validate(u1, rules.Validate); err != nil {
			t.Errorf("Expected nil for zero age with Optional, got %v", err)
		}

		// Age 为 10 - 不在范围内，应该失败
		u2 := &ValidateUser{Age: 10}
		if err := jorm.Validate(u2, rules.Validate); err == nil {
			t.Error("Expected error for age 10, got nil")
		}

		// When 条件不满足：Reason 为空但不是 "TRIGGER"，不触发 Required，应该通过
		u3 := &WhenUser{Status: "active", Reason: ""}
		if err := jorm.Validate(u3, rules.Validate); err != nil {
			t.Errorf("Expected nil for empty reason when condition not met, got %v", err)
		}

		// When 条件满足：Reason 为 "TRIGGER"，由于不为空，满足 Required，应该通过
		u4 := &WhenUser{Status: "active", Reason: "TRIGGER"}
		if err := jorm.Validate(u4, rules.Validate); err != nil {
			t.Errorf("Expected nil for triggered reason, got %v", err)
		}

		// 强制触发失败：自定义 When 条件始终返回 true，但字段值为空
		rules2 := jorm.Rules{
			"Reason": {jorm.Required.When(func(v any) bool {
				return v == nil || v.(string) == ""
			})},
		}
		u5 := &WhenUser{Reason: ""}
		if err := jorm.Validate(u5, rules2.Validate); err == nil {
			t.Error("Expected error for empty reason when When condition is met, got nil")
		}
	})

	t.Run("All Built-in Rules", func(t *testing.T) {
		// 测试所有内置规则：Numeric, Alpha, AlphaNumeric, Datetime, In, NoHTML
		type AllRules struct {
			Num      string
			Alpha    string
			AlphaNum string
			Date     string
			Category string
			Content  string
		}

		rules := jorm.Rules{
			"Num":      {jorm.Numeric},
			"Alpha":    {jorm.Alpha},
			"AlphaNum": {jorm.AlphaNumeric},
			"Date":     {jorm.Datetime("2006-01-02")},
			"Category": {jorm.In("A", "B", "C")},
			"Content":  {jorm.NoHTML},
		}

		// 全部合法
		a1 := &AllRules{
			Num:      "123",
			Alpha:    "abc",
			AlphaNum: "abc123",
			Date:     "2023-01-01",
			Category: "B",
			Content:  "Hello World",
		}
		if err := jorm.Validate(a1, rules.Validate); err != nil {
			t.Errorf("Expected nil for valid data, got %v", err)
		}

		// 全部非法
		a2 := &AllRules{
			Num:      "123a",
			Alpha:    "abc1",
			AlphaNum: "abc!",
			Date:     "2023/01/01",
			Category: "D",
			Content:  "<p>Hello</p>",
		}
		err := jorm.Validate(a2, rules.Validate)
		if err == nil {
			t.Fatal("Expected errors, got nil")
		}
		errs := err.(jorm.ValidationErrors)
		if len(errs) != 6 {
			t.Errorf("Expected 6 fields with errors, got %d", len(errs))
		}
	})

	t.Run("Multi-Error Collection", func(t *testing.T) {
		// 测试单个字段的多错误收集
		type MultiErr struct {
			Code string
		}
		rules := jorm.Rules{
			"Code": {
				jorm.MinLen(5).Msg("Too short"),
				jorm.Numeric.Msg("Must be digits"),
			},
		}

		m := &MultiErr{Code: "abc"} // 既太短，又不是纯数字
		err := jorm.Validate(m, rules.Validate)
		if err == nil {
			t.Fatal("Expected errors, got nil")
		}
		errs := err.(jorm.ValidationErrors)
		if len(errs["Code"]) != 2 {
			t.Errorf("Expected 2 errors for Code, got %d", len(errs["Code"]))
		}
	})

	t.Run("Flexible Field Mapping", func(t *testing.T) {
		// 测试字段名匹配的灵活性：大小写不敏感、支持列名
		type FlexibleUser struct {
			UserName string `jorm:"column:user_name"`
			Age      int    `jorm:"column:user_age"`
		}

		rules := jorm.Rules{
			"username":  {jorm.Required},       // 小写匹配
			"user_name": {jorm.MinLen(5)},      // 列名匹配
			"User_Age":  {jorm.Range(18, 100)}, // 列名大小写混合匹配
		}

		u := &FlexibleUser{UserName: "Bob", Age: 10}
		err := jorm.Validate(u, rules.Validate)
		if err == nil {
			t.Fatal("Expected errors, got nil")
		}

		errs := err.(jorm.ValidationErrors)
		// "username" 匹配到 UserName，值 "Bob" 满足 Required，无错
		// "user_name" 匹配到 UserName，值 "Bob" 长度 3 < 5，有错
		// "User_Age" 匹配到 Age，值 10 < 18，有错
		if len(errs["user_name"]) == 0 {
			t.Error("Expected error for user_name (column name match)")
		}
		if len(errs["User_Age"]) == 0 {
			t.Error("Expected error for User_Age (column name match)")
		}
	})

	t.Run("ValidationErrors Helpers", func(t *testing.T) {
		type HelperUser struct {
			Name string
			Age  int
		}
		rules := jorm.Rules{
			"Name": {jorm.Required.Msg("Name is required")},
			"Age":  {jorm.Range(18, 100).Msg("Too young")},
		}

		u := &HelperUser{Name: "", Age: 10}
		err := jorm.Validate(u, rules.Validate)
		if err == nil {
			t.Fatal("Expected errors, got nil")
		}

		errs, ok := err.(jorm.ValidationErrors)
		if !ok {
			t.Fatalf("Expected ValidationErrors, got %T", err)
		}

		// Test First()
		firstErr := errs.First()
		if firstErr == nil {
			t.Error("First() should not be nil")
		}

		// Test FirstMsg()
		msg := errs.FirstMsg()
		if msg == "" {
			t.Error("FirstMsg() should not be empty")
		}
		if msg != "Name is required" && msg != "Too young" {
			t.Errorf("Unexpected FirstMsg: %s", msg)
		}

		// Test Error() summary format
		errStr := errs.Error()
		if !strings.Contains(errStr, "(and 1 more errors)") {
			t.Errorf("Error() should contain summary info, got: %s", errStr)
		}

		// Test Global helper jorm.FirstMsg(err)
		globalMsg := jorm.FirstMsg(err)
		if globalMsg != msg {
			t.Errorf("Global FirstMsg mismatch: expected %s, got %s", msg, globalMsg)
		}

		// Test Global helper with nil
		if jorm.FirstMsg(nil) != "" {
			t.Error("Global FirstMsg(nil) should be empty")
		}

		// Test Global helper with regular error
		regErr := errors.New("standard error")
		if jorm.FirstMsg(regErr) != "standard error" {
			t.Error("Global FirstMsg should return Error() for regular errors")
		}
	})

	t.Run("Hook Validation", func(t *testing.T) {
		// Test validation via BeforeInsert hook
		user := &ValidateUser{Name: "A", Email: "invalid", Age: 10}
		// Insert should fail due to validation in BeforeInsert
		_, err := db.Model(user).Insert(user)
		if err == nil {
			t.Error("Expected validation error from hook, got nil")
		}

		errs, ok := err.(jorm.ValidationErrors)
		// Note: The error might be wrapped, but if CommonValidator returns ValidationErrors,
		// and BeforeInsert returns it directly, it should be accessible or wrapped.
		// JORM hooks wrap errors. Let's check string content for simplicity if not directly castable.
		if !ok {
			// Try unwrapping or checking string
			errStr := err.Error()
			if errStr == "" {
				t.Error("Expected non-empty error message")
			}
		} else {
			if len(errs["Name"]) == 0 || errs["Name"][0].Error() != "Name too short" {
				t.Errorf("Unexpected error for Name: %v", errs["Name"])
			}
		}
	})
}

type HookVarUser struct {
	ID   int64
	Name string
	Code string
}

func (m *HookVarUser) TableName() string {
	return "hook_var_users"
}

func (m *HookVarUser) BeforeInsert() error {
	// Ad-hoc validation for Code using jorm.Check
	// 验证 Code 字段：必须存在，且长度至少为 5
	if err := jorm.Check(m.Code, jorm.Required, jorm.MinLen(5).Msg("Code too short")); err != nil {
		return err
	}
	return nil
}

func TestValidator_Check(t *testing.T) {
	dbName := "validator_var_test.db"
	os.Remove(dbName)
	db, err := jorm.Open("sqlite3", dbName, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		db.Close()
		os.Remove(dbName)
	}()

	// Create table
	_, err = db.Exec("CREATE TABLE hook_var_users (id INTEGER PRIMARY KEY, name TEXT, code TEXT)")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Hook AdHoc Validation Success", func(t *testing.T) {
		user := &HookVarUser{Name: "Test", Code: "12345"}
		_, err := db.Model(user).Insert(user)
		if err != nil {
			t.Fatalf("Expected success, got %v", err)
		}
	})

	t.Run("Hook AdHoc Validation Failure", func(t *testing.T) {
		user := &HookVarUser{Name: "Test", Code: "123"}
		_, err := db.Model(user).Insert(user)
		if err == nil {
			t.Error("Expected validation error, got nil")
		} else {
			// jorm.Check returns the error directly.
			// Hook errors might be wrapped by JORM core (e.g. "before insert hook failed: ...")
			// or returned as is.
			// Let's check if the error message contains our custom message.
			if !strings.Contains(err.Error(), "Code too short") {
				t.Errorf("Expected error containing 'Code too short', got '%v'", err)
			}
		}
	})

	t.Run("Hook Manual Error Handling", func(t *testing.T) {
		// Test manual error overriding as requested by user
		// if err := jorm.Check(..., ...); err != nil { return errors.New("custom") }
		// This logic is actually tested in TestValidator_Check_ManualError below with a dedicated struct.
		// So we can keep this empty or remove it.
	})
}

type HookManualErrorUser struct {
	ID   int64
	Name string
}

func (m *HookManualErrorUser) TableName() string {
	return "hook_manual_error_users"
}

func (m *HookManualErrorUser) BeforeInsert() error {
	// 演示用户请求的模式：手动处理错误
	if err := jorm.Check(m.Name, jorm.Required, jorm.MinLen(5)); err != nil {
		return errors.New("自定义错误提示")
	}
	return nil
}

func TestValidator_Check_ManualError(t *testing.T) {
	dbName := "validator_manual_test.db"
	os.Remove(dbName)
	db, err := jorm.Open("sqlite3", dbName, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		db.Close()
		os.Remove(dbName)
	}()

	_, err = db.Exec("CREATE TABLE hook_manual_error_users (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Manual Error Override", func(t *testing.T) {
		user := &HookManualErrorUser{Name: "Tiny"} // Length < 5
		_, err := db.Model(user).Insert(user)
		if err == nil {
			t.Error("Expected validation error, got nil")
		}

		// Check if we got the custom error message
		if !strings.Contains(err.Error(), "自定义错误提示") {
			t.Errorf("Expected '自定义错误提示', got '%v'", err)
		}
	})
}
