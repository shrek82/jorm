package tests

import (
	"testing"

	"github.com/shrek82/jorm/model"
)

type TestUser struct {
	ID        int64  `jorm:"pk auto column:id"`
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
