package tests

import (
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shrek82/jorm/core"
)

type MigrationUser struct {
	ID   int64  `jorm:"pk;auto"`
	Name string `jorm:"column:name"`
}

type MigrationUserV2 struct {
	ID    int64  `jorm:"pk;auto"`
	Name  string `jorm:"column:name"`
	Email string `jorm:"column:email"`
}

func TestAutoMigrate(t *testing.T) {
	dbFile := "migration_test.db"
	defer os.Remove(dbFile)

	db, err := core.Open("sqlite3", dbFile, nil)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// 1. Initial migration
	err = db.AutoMigrate(&MigrationUser{})
	if err != nil {
		t.Fatalf("AutoMigrate V1 failed: %v", err)
	}

	exists, err := db.HasTable("migration_user")
	if err != nil || !exists {
		t.Errorf("Table migration_user should exist")
	}

	// 2. Migration with new column
	err = db.AutoMigrate(&MigrationUserV2{})
	if err != nil {
		t.Fatalf("AutoMigrate V2 failed: %v", err)
	}

	// Verify column exists
	// We can try to insert a record with the new column
	user := &MigrationUserV2{Name: "test", Email: "test@example.com"}
	_, err = db.Model(user).Insert(user)
	if err != nil {
		t.Errorf("Failed to insert user with new column: %v", err)
	}
}

func TestMigrator(t *testing.T) {
	dbFile := "migrator_test.db"
	defer os.Remove(dbFile)

	db, err := core.Open("sqlite3", dbFile, nil)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	migrator := core.NewMigrator(db)

	m1 := &core.Migration{
		Version:     1,
		Description: "Create users table",
		Up: func(db *core.DB) error {
			_, err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
			return err
		},
		Down: func(db *core.DB) error {
			_, err := db.Exec("DROP TABLE users")
			return err
		},
	}

	// Apply migration
	err = migrator.Migrate(m1)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	exists, _ := db.HasTable("users")
	if !exists {
		t.Errorf("Table users should exist")
	}

	// Rollback
	err = migrator.Rollback(m1)
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	exists, _ = db.HasTable("users")
	if exists {
		t.Errorf("Table users should not exist after rollback")
	}
}
