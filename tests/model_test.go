package tests

import (
	"testing"
	"time"

	"github.com/shrek82/jorm/model"
)

type TestUser struct {
	ID        int64  `jorm:"pk;auto column:id"`
	UserName  string `jorm:"column:user_name size:100"`
	Email     string `jorm:"unique"`
	Age       int
	IgnoreMe  string `jorm:"-"`
	CreatedAt int64
}

type EmbeddedUser struct {
	TestUser
	ExtraInfo string
}

func TestGetModel(t *testing.T) {
	t.Run("BasicModel", func(t *testing.T) {
		m, err := model.GetModel(&TestUser{})
		if err != nil {
			t.Fatalf("Failed to get model: %v", err)
		}

		if m.TableName != "test_user" {
			t.Errorf("Expected table name 'test_user', got '%s'", m.TableName)
		}

		if len(m.Fields) != 5 { // ID, UserName, Email, Age, CreatedAt (IgnoreMe is ignored)
			t.Errorf("Expected 5 fields, got %d", len(m.Fields))
		}

		// Check PK
		if m.PKField == nil || m.PKField.Name != "ID" {
			t.Errorf("Expected ID as PK field")
		}
		if m.PKField.IsPK != true {
			t.Errorf("Expected PKField.IsPK to be true")
		}
		if m.PKField.IsAuto != true {
			t.Errorf("Expected PKField.IsAuto to be true")
		}
		if m.PKField.Column != "id" {
			t.Errorf("Expected column name 'id', got '%s'", m.PKField.Column)
		}

		// Check FieldMap
		if _, ok := m.FieldMap["user_name"]; !ok {
			t.Errorf("Field 'user_name' should exist in FieldMap")
		}
		if _, ok := m.FieldMap["email"]; !ok {
			t.Errorf("Field 'email' should exist in FieldMap")
		}
	})

	t.Run("EmbeddedModel", func(t *testing.T) {
		m, err := model.GetModel(&EmbeddedUser{})
		if err != nil {
			t.Fatalf("Failed to get model: %v", err)
		}

		if m.TableName != "embedded_user" {
			t.Errorf("Expected table name 'embedded_user', got '%s'", m.TableName)
		}

		// 5 fields from TestUser + 1 from EmbeddedUser
		if len(m.Fields) != 6 {
			t.Errorf("Expected 6 fields, got %d", len(m.Fields))
		}

		if _, ok := m.FieldMap["id"]; !ok {
			t.Errorf("Embedded field 'id' should exist")
		}
		if _, ok := m.FieldMap["extra_info"]; !ok {
			t.Errorf("Field 'extra_info' should exist")
		}
	})

	t.Run("InvalidModel", func(t *testing.T) {
		_, err := model.GetModel(123)
		if err == nil {
			t.Errorf("Expected error for non-struct type, got nil")
		}

		_, err = model.GetModel(nil)
		if err == nil {
			t.Errorf("Expected error for nil value, got nil")
		}
	})
}

func TestModelCache(t *testing.T) {
	m1, _ := model.GetModel(&TestUser{})
	m2, _ := model.GetModel(&TestUser{})

	if m1 != m2 {
		t.Errorf("Model metadata should be cached and return same pointer")
	}
}

func TestModelValidation(t *testing.T) {
	t.Run("InvalidAutoTime", func(t *testing.T) {
		type InvalidAutoTime struct {
			CreatedAt string `jorm:"auto_time"`
		}
		_, err := model.GetModel(&InvalidAutoTime{})
		if err == nil {
			t.Error("Expected error for string field with auto_time, got nil")
		}
	})

	t.Run("InvalidAutoUpdate", func(t *testing.T) {
		type InvalidAutoUpdate struct {
			UpdatedAt int64 `jorm:"auto_update"`
		}
		_, err := model.GetModel(&InvalidAutoUpdate{})
		if err == nil {
			t.Error("Expected error for int64 field with auto_update, got nil")
		}
	})

	t.Run("InvalidAuto", func(t *testing.T) {
		type InvalidAuto struct {
			ID string `jorm:"auto"`
		}
		_, err := model.GetModel(&InvalidAuto{})
		if err == nil {
			t.Error("Expected error for string field with auto, got nil")
		}
	})

	t.Run("InvalidAutoFloat", func(t *testing.T) {
		type InvalidAutoFloat struct {
			ID float64 `jorm:"auto"`
		}
		_, err := model.GetModel(&InvalidAutoFloat{})
		if err == nil {
			t.Error("Expected error for float64 field with auto, got nil")
		}
	})

	t.Run("ValidTypes", func(t *testing.T) {
		type ValidModel struct {
			ID        int64      `jorm:"auto"`
			IDPtr     *int       `jorm:"auto"`
			CreatedAt time.Time  `jorm:"auto_time"`
			UpdatedAt *time.Time `jorm:"auto_update"`
		}
		_, err := model.GetModel(&ValidModel{})
		if err != nil {
			t.Errorf("Expected valid model to pass validation, got error: %v", err)
		}
	})
}
