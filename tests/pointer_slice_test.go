package tests

import (
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type PtrSliceUser struct {
	ID        int64     `jorm:"pk auto"`
	Name      string    `jorm:"size:100"`
	CreatedAt time.Time `jorm:"auto_time"`
}

func TestFindWithPointerSlice(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := db.AutoMigrate(&PtrSliceUser{})
	if err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	// Insert some users
	users := []*PtrSliceUser{
		{Name: "User1"},
		{Name: "User2"},
	}
	_, err = db.Model(&PtrSliceUser{}).BatchInsert(users)
	if err != nil {
		t.Fatalf("BatchInsert failed: %v", err)
	}

	// Test Find with *[]*User
	var resultPtrs []*PtrSliceUser
	err = db.Model(&PtrSliceUser{}).OrderBy("id ASC").Find(&resultPtrs)
	if err != nil {
		t.Fatalf("Find with *[]*User failed: %v", err)
	}

	if len(resultPtrs) != 2 {
		t.Errorf("Expected 2 users, got %d", len(resultPtrs))
	}
	if resultPtrs[0].Name != "User1" {
		t.Errorf("Expected User1, got %s", resultPtrs[0].Name)
	}
	if resultPtrs[1].Name != "User2" {
		t.Errorf("Expected User2, got %s", resultPtrs[1].Name)
	}

	// Test Find with *[]User (Regression check)
	var resultStructs []PtrSliceUser
	err = db.Model(&PtrSliceUser{}).OrderBy("id ASC").Find(&resultStructs)
	if err != nil {
		t.Fatalf("Find with *[]User failed: %v", err)
	}

	if len(resultStructs) != 2 {
		t.Errorf("Expected 2 users, got %d", len(resultStructs))
	}
	if resultStructs[0].Name != "User1" {
		t.Errorf("Expected User1, got %s", resultStructs[0].Name)
	}
}
