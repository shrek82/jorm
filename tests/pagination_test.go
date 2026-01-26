package tests

import (
	"fmt"
	"testing"
	"time"
)

type PaginationUser struct {
	ID        int64     `jorm:"pk auto"`
	Name      string    `jorm:"size:100"`
	CreatedAt time.Time `jorm:"auto_time"`
}

func TestPaginate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := db.AutoMigrate(&PaginationUser{})
	if err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	// Insert 25 users
	var userPtrs []*PaginationUser
	for i := 1; i <= 25; i++ {
		userPtrs = append(userPtrs, &PaginationUser{Name: fmt.Sprintf("User%d", i)})
	}

	_, err = db.Model(&PaginationUser{}).BatchInsert(userPtrs)
	if err != nil {
		t.Fatalf("BatchInsert failed: %v", err)
	}

	// Test Page 1, PerPage 10
	var page1Users []*PaginationUser
	pagination, err := db.Model(&PaginationUser{}).OrderBy("id ASC").Paginate(1, 10, &page1Users)
	if err != nil {
		t.Fatalf("Paginate page 1 failed: %v", err)
	}

	if pagination.ItemTotal != 25 {
		t.Errorf("Expected ItemTotal 25, got %d", pagination.ItemTotal)
	}
	if pagination.TotalPage != 3 {
		t.Errorf("Expected TotalPage 3, got %d", pagination.TotalPage)
	}
	if pagination.Page != 1 {
		t.Errorf("Expected Page 1, got %d", pagination.Page)
	}
	if pagination.PerPage != 10 {
		t.Errorf("Expected PerPage 10, got %d", pagination.PerPage)
	}
	if len(page1Users) != 10 {
		t.Errorf("Expected 10 users on page 1, got %d", len(page1Users))
	}
	if len(page1Users) > 0 && page1Users[0].Name != "User1" {
		t.Errorf("Expected first user to be User1, got %s", page1Users[0].Name)
	}

	// Test Page 3, PerPage 10 (Last page)
	var page3Users []*PaginationUser
	pagination3, err := db.Model(&PaginationUser{}).OrderBy("id ASC").Paginate(3, 10, &page3Users)
	if err != nil {
		t.Fatalf("Paginate page 3 failed: %v", err)
	}

	if pagination3.Page != 3 {
		t.Errorf("Expected Page 3, got %d", pagination3.Page)
	}
	if len(page3Users) != 5 {
		t.Errorf("Expected 5 users on page 3, got %d", len(page3Users))
	}
	if len(page3Users) > 0 && page3Users[0].Name != "User21" {
		t.Errorf("Expected first user on page 3 to be User21, got %s", page3Users[0].Name)
	}

	// Test with conditions
	// Find users with ID > 20 (5 users: 21, 22, 23, 24, 25)
	var filteredUsers []*PaginationUser
	paginationFiltered, err := db.Model(&PaginationUser{}).Where("id > ?", 20).Paginate(1, 2, &filteredUsers)
	if err != nil {
		t.Fatalf("Paginate filtered failed: %v", err)
	}

	if paginationFiltered.ItemTotal != 5 {
		t.Errorf("Expected filtered ItemTotal 5, got %d", paginationFiltered.ItemTotal)
	}
	if paginationFiltered.TotalPage != 3 { // 5 items, 2 per page -> 3 pages
		t.Errorf("Expected filtered TotalPage 3, got %d", paginationFiltered.TotalPage)
	}
	if len(filteredUsers) != 2 {
		t.Errorf("Expected 2 filtered users on page 1, got %d", len(filteredUsers))
	}
}
