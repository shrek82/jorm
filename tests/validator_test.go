package tests

import (
	"errors"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shrek82/jorm"
)

type ValidateUser struct {
	ID    int64  `jorm:"pk,auto"`
	Name  string `jorm:"column:name"`
	Email string `jorm:"column:email"`
	Age   int    `jorm:"column:age"`
	Role  string `jorm:"column:role"`
}

func (m *ValidateUser) TableName() string {
	return "validate_users"
}

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

func (m *ValidateUser) AdminValidator() jorm.Validator {
	return func(value any) error {
		u := value.(*ValidateUser)
		if u.Role != "admin" {
			return errors.New("Unauthorized")
		}
		return nil
	}
}

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

func teardownValidatorDB() {
	os.Remove("validator_test.db")
}

func TestValidator(t *testing.T) {
	db := setupValidatorDB(t)
	defer teardownValidatorDB()

	t.Run("Standalone Validation", func(t *testing.T) {
		user := &ValidateUser{Name: "A", Email: "invalid", Age: 10}
		err := jorm.Validate(user, user.CommonValidator())
		if err == nil {
			t.Error("Expected validation error, got nil")
		}

		errs, ok := err.(jorm.ValidationErrors)
		if !ok {
			t.Errorf("Expected ValidationErrors, got %T", err)
		}

		if len(errs["Name"]) == 0 || errs["Name"][0].Error() != "Name too short" {
			t.Errorf("Unexpected error for Name: %v", errs["Name"])
		}
		if len(errs["Email"]) == 0 || errs["Email"][0].Error() != "Invalid email" {
			t.Errorf("Unexpected error for Email: %v", errs["Email"])
		}
	})

	t.Run("InsertWithValidator - Success", func(t *testing.T) {
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
		user := &ValidateUser{Name: "", Email: "bad", Age: 5}
		_, err := db.Model(user).InsertWithValidator(user, user.CommonValidator())
		if err == nil {
			t.Error("Expected validation error, got nil")
		}
	})

	t.Run("Combination Validator", func(t *testing.T) {
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
		user := &ValidateUser{Name: "Initial", Email: "init@example.com", Age: 20}
		id, _ := db.Model(user).Insert(user)
		user.ID = id

		// Valid update
		user.Name = "Updated Name"
		affected, err := db.Model(user).Where("id = ?", id).UpdateWithValidator(user, user.CommonValidator())
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if affected == 0 {
			t.Error("Expected affected rows > 0")
		}

		// Invalid update
		user.Email = "invalid-email"
		_, err = db.Model(user).Where("id = ?", id).UpdateWithValidator(user, user.CommonValidator())
		if err == nil {
			t.Error("Expected validation error for invalid email update, got nil")
		}
	})

	t.Run("Extended Format Rules", func(t *testing.T) {
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

		// Valid data
		m1 := &MetaData{
			IP:   "192.168.1.1",
			JSON: `{"key": "value"}`,
			UUID: "550e8400-e29b-41d4-a716-446655440000",
			Tags: "golang-is-great",
		}
		if err := jorm.Validate(m1, rules.Validate); err != nil {
			t.Errorf("Expected nil for valid metadata, got %v", err)
		}

		// Invalid data
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
		type WhenUser struct {
			Status string
			Reason string
		}

		rules := jorm.Rules{
			"Age": {jorm.Range(18, 100).Optional()},
			"Reason": {jorm.Required.When(func(v any) bool {
				// This is tricky because When(v) receives the value of the field (Reason),
				// but often we need other fields. For complex cross-field validation,
				// users should use a custom validator function.
				// However, let's test the current When implementation.
				return v != nil && v.(string) == "TRIGGER"
			})},
		}

		// Empty Age (0) - should pass because of Optional
		u1 := &ValidateUser{Age: 0}
		if err := jorm.Validate(u1, rules.Validate); err != nil {
			t.Errorf("Expected nil for zero age with Optional, got %v", err)
		}

		// Invalid Age (10) - should fail
		u2 := &ValidateUser{Age: 10}
		if err := jorm.Validate(u2, rules.Validate); err == nil {
			t.Error("Expected error for age 10, got nil")
		}

		// When condition
		u3 := &WhenUser{Status: "active", Reason: ""}
		if err := jorm.Validate(u3, rules.Validate); err != nil {
			t.Errorf("Expected nil for empty reason when condition not met, got %v", err)
		}

		u4 := &WhenUser{Status: "active", Reason: "TRIGGER"}
		if err := jorm.Validate(u4, rules.Validate); err != nil {
			t.Errorf("Expected nil for triggered reason, got %v", err)
		}

		// Failure case for When
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

		// Valid
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

		// Invalid
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
		type MultiErr struct {
			Code string
		}
		rules := jorm.Rules{
			"Code": {
				jorm.MinLen(5).Msg("Too short"),
				jorm.Numeric.Msg("Must be digits"),
			},
		}

		m := &MultiErr{Code: "abc"}
		err := jorm.Validate(m, rules.Validate)
		if err == nil {
			t.Fatal("Expected errors, got nil")
		}
		errs := err.(jorm.ValidationErrors)
		if len(errs["Code"]) != 2 {
			t.Errorf("Expected 2 errors for Code, got %d", len(errs["Code"]))
		}
	})
}
