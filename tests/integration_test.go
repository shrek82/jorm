package tests

import (
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shrek82/jorm/core"
	_ "github.com/shrek82/jorm/dialect"
)

type User struct {
	ID        int64     `jorm:"pk auto"`
	Name      string    `jorm:"size:100 notnull"`
	Email     string    `jorm:"size:100 unique"`
	Age       int       `jorm:"default:0"`
	CreatedAt time.Time `jorm:"auto_time"`
	UpdatedAt time.Time `jorm:"auto_update"`
}

// Hooks for User
func (u *User) BeforeInsert() error {
	fmt.Println("BeforeInsert hook called")
	return nil
}

func (u *User) AfterFind() error {
	fmt.Println("AfterFind hook called")
	return nil
}

func TestIntegration(t *testing.T) {
	dbFile := "test.db"
	defer os.Remove(dbFile)

	db, err := core.Open("sqlite3", dbFile, &core.Options{
		MaxOpenConns: 1,
	})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// 1. AutoMigrate
	err = db.AutoMigrate(&User{})
	if err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	// 2. Insert
	user := &User{
		Name:  "Alice",
		Email: "alice@example.com",
		Age:   25,
	}
	id, err := db.Model(user).Insert(user)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if id == 0 {
		t.Fatal("Insert ID should not be 0")
	}

	// 3. Find One
	var alice User
	err = db.Model(&User{}).Where("id = ?", id).First(&alice)
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}
	if alice.Name != "Alice" {
		t.Errorf("Expected name Alice, got %s", alice.Name)
	}

	// 4. Update
	alice.Age = 26
	affected, err := db.Model(&alice).Where("id = ?", id).Update(&alice)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if affected != 1 {
		t.Errorf("Expected 1 row affected, got %d", affected)
	}

	// 5. Find All
	var users []User
	err = db.Model(&User{}).Find(&users)
	if err != nil {
		t.Fatalf("Find all failed: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(users))
	}

	// 6. Delete
	affected, err = db.Model(&User{}).Where("id = ?", id).Delete()
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if affected != 1 {
		t.Errorf("Expected 1 row affected by delete, got %d", affected)
	}

	// 7. Verify deletion
	var deletedUser User
	err = db.Model(&User{}).Where("id = ?", id).Find(&deletedUser)
	if err == nil {
		t.Error("Expected error finding deleted user, but got none")
	}

	// 8. Transaction
	err = db.Transaction(func(tx *core.Tx) error {
		user1 := &User{Name: "TxUser1", Email: "tx1@example.com"}
		_, err := tx.Model(user1).Insert(user1)
		if err != nil {
			return err
		}

		user2 := &User{Name: "TxUser2", Email: "tx2@example.com"}
		_, err = tx.Model(user2).Insert(user2)
		if err != nil {
			return err
		}
		return nil // Commit
	})
	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	var txUsers []User
	err = db.Model(&User{}).Where("name LIKE ?", "TxUser%").Find(&txUsers)
	if err != nil {
		t.Fatalf("Find tx users failed: %v", err)
	}
	if len(txUsers) != 2 {
		t.Errorf("Expected 2 tx users, got %d", len(txUsers))
	}

	// 9. Transaction Rollback
	err = db.Transaction(func(tx *core.Tx) error {
		user3 := &User{Name: "TxUser3", Email: "tx3@example.com"}
		_, err := tx.Model(user3).Insert(user3)
		if err != nil {
			return err
		}
		return fmt.Errorf("trigger rollback")
	})
	if err == nil || err.Error() != "trigger rollback" {
		t.Fatalf("Expected rollback error, got %v", err)
	}

	var txUser3 User
	err = db.Model(&User{}).Where("name = ?", "TxUser3").First(&txUser3)
	if err == nil {
		t.Error("Expected TxUser3 to not exist due to rollback")
	}

	// 10. Batch Insert
	batchUsers := []*User{
		{Name: "Batch1", Email: "b1@example.com"},
		{Name: "Batch2", Email: "b2@example.com"},
	}
	count, err := db.Model(&User{}).BatchInsert(batchUsers)
	if err != nil {
		t.Fatalf("BatchInsert failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 rows affected by BatchInsert, got %d", count)
	}
}
