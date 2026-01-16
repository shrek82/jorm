package tests

import (
	"strings"
	"testing"

	"github.com/shrek82/jorm"
)

// UserWithHook demonstrates how to integrate validation with hooks.
type UserWithHook struct {
	ID    int64  `jorm:"pk auto"`
	Name  string `jorm:"size:50"`
	Email string `jorm:"size:100"`
	Age   int
}

// Validate implements the validation logic using jorm.Rules.
// This keeps the validation logic centralized and declarative.
func (u *UserWithHook) Validate() error {
	// Define rules
	rules := jorm.Rules{
		"Name": {
			jorm.Required.Msg("Name is required"),
			jorm.MinLen(2).Msg("Name must be at least 2 chars"),
		},
		"Email": {
			jorm.Required,
			jorm.Email.Msg("Invalid email format"),
		},
		"Age": {
			jorm.Range(18, 150).Msg("Age must be between 18 and 150"),
		},
	}

	// Execute validation
	return rules.Validate(u)
}

// BeforeInsert hook calls Validate.
func (u *UserWithHook) BeforeInsert() error {
	return u.Validate()
}

// BeforeUpdate hook calls Validate.
func (u *UserWithHook) BeforeUpdate() error {
	return u.Validate()
}

// TestHookValidation_Success 测试通过钩子进行验证的成功场景
// 步骤：
// 1. 创建符合规则的 UserWithHook 对象
// 2. 调用 Insert
// 3. 验证插入成功（无错误）
func TestHookValidation_Success(t *testing.T) {
	db := setupValidatorDB(t)
	defer teardownValidatorDB()

	// Ensure table exists (UserWithHook needs migration)
	if err := db.AutoMigrate(&UserWithHook{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	user := &UserWithHook{
		Name:  "Alice",
		Email: "alice@example.com",
		Age:   25,
	}

	_, err := db.Model(user).Insert(user)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
}

// TestHookValidation_Failure 测试通过钩子进行验证的失败场景
// 步骤：
// 1. 创建不符合规则的 UserWithHook 对象（Name为空，Age不合法）
// 2. 调用 Insert
// 3. 验证返回错误，且错误包含预期的验证信息
func TestHookValidation_Failure(t *testing.T) {
	db := setupValidatorDB(t)
	defer teardownValidatorDB()

	if err := db.AutoMigrate(&UserWithHook{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	// Invalid user
	user := &UserWithHook{
		Name:  "", // Required
		Email: "invalid-email",
		Age:   10, // Min 18
	}

	_, err := db.Model(user).Insert(user)
	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}

	// Check error message
	// The error returned by BeforeInsert is wrapped
	errStr := err.Error()
	if !strings.Contains(errStr, "Name is required") {
		t.Errorf("Expected 'Name is required' in error: %s", errStr)
	}
	if !strings.Contains(errStr, "Invalid email format") {
		t.Errorf("Expected 'Invalid email format' in error: %s", errStr)
	}
	if !strings.Contains(errStr, "Age must be between 18 and 150") {
		t.Errorf("Expected 'Age must be between 18 and 150' in error: %s", errStr)
	}
}
