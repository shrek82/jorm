package tests

import (
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/shrek82/jorm/core"
)

func setupPostgresTestDB(t *testing.T) (*core.DB, func()) {
	t.Helper()

	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN not set, skipping Postgres tests")
	}

	db, err := core.Open("postgres", dsn, &core.Options{
		MaxOpenConns: 5,
		MaxIdleConns: 5,
	})
	if err != nil {
		t.Fatalf("failed to open Postgres: %v", err)
	}

	err = db.AutoMigrate(&User{})
	if err != nil {
		db.Close()
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	_, err = db.Model(&User{}).Where("1 = 1").Delete()
	if err != nil {
		db.Close()
		t.Fatalf("failed to clean user table: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func TestPostgresIntegration(t *testing.T) {
	t.Run("Connect", func(t *testing.T) {
		db, cleanup := setupPostgresTestDB(t)
		defer cleanup()

		if db == nil {
			t.Fatalf("db should not be nil")
		}
	})

	t.Run("CRUD", func(t *testing.T) {
		db, cleanup := setupPostgresTestDB(t)
		defer cleanup()

		user := &User{
			Name:  "PGUser",
			Email: "pg@example.com",
			Age:   30,
		}
		_, err := db.Model(user).Insert(user)
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}

		var got User
		err = db.Model(&User{}).Where("email = ?", user.Email).First(&got)
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}
		if got.Name != "PGUser" {
			t.Fatalf("Expected name PGUser, got %s", got.Name)
		}

		_, err = db.Model(&User{}).Where("email = ?", user.Email).Update(map[string]any{"age": 31})
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		var updated User
		err = db.Model(&User{}).Where("email = ?", user.Email).First(&updated)
		if err != nil {
			t.Fatalf("Find after update failed: %v", err)
		}
		if updated.Age != 31 {
			t.Fatalf("Expected age 31, got %d", updated.Age)
		}

		_, err = db.Model(&User{}).Where("email = ?", user.Email).Delete()
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		count, err := db.Model(&User{}).Where("email = ?", user.Email).Count()
		if err != nil {
			t.Fatalf("Count failed: %v", err)
		}
		if count != 0 {
			t.Fatalf("Expected count 0 after delete, got %d", count)
		}
	})

	t.Run("BatchInsertAndQueryHelpers", func(t *testing.T) {
		db, cleanup := setupPostgresTestDB(t)
		defer cleanup()

		prefix := fmt.Sprintf("PGBatchUser_%d_", time.Now().UnixNano())

		var users []User
		for i := 0; i < 5; i++ {
			now := time.Now()
			users = append(users, User{
				Name:      fmt.Sprintf("%s%d", prefix, i+1),
				Email:     fmt.Sprintf("%s%d@example.com", prefix, i+1),
				Age:       20 + i,
				CreatedAt: now,
				UpdatedAt: now,
			})
		}

		_, err := db.Model(&User{}).BatchInsert(users)
		if err != nil {
			t.Fatalf("BatchInsert failed: %v", err)
		}

		var result []User
		err = db.Model(&User{}).
			Where("email LIKE ?", prefix+"%").
			OrderBy("age DESC").
			Limit(3).
			Find(&result)
		if err != nil {
			t.Fatalf("Query with helpers failed: %v", err)
		}
		if len(result) != 3 {
			t.Fatalf("Expected 3 users, got %d", len(result))
		}
		if result[0].Age < result[1].Age || result[1].Age < result[2].Age {
			t.Fatalf("Users are not ordered by age DESC")
		}

		count, err := db.Model(&User{}).
			Where("email LIKE ?", prefix+"%").
			Count()
		if err != nil {
			t.Fatalf("Count failed: %v", err)
		}
		if count != 5 {
			t.Fatalf("Expected count 5, got %d", count)
		}

		sum, err := db.Model(&User{}).
			Where("email LIKE ?", prefix+"%").
			Sum("age")
		if err != nil {
			t.Fatalf("Sum failed: %v", err)
		}
		if sum <= 0 {
			t.Fatalf("Expected positive sum of ages, got %v", sum)
		}
	})

	t.Run("TransactionCommitAndRollback", func(t *testing.T) {
		db, cleanup := setupPostgresTestDB(t)
		defer cleanup()

		commitEmail := fmt.Sprintf("pg_tx_commit_%d@example.com", time.Now().UnixNano())
		err := db.Transaction(func(tx *core.Tx) error {
			_, err := tx.Model(&User{}).Insert(&User{
				Name:  "PGTxCommitUser",
				Email: commitEmail,
				Age:   40,
			})
			return err
		})
		if err != nil {
			t.Fatalf("Transaction commit failed: %v", err)
		}

		commitCount, err := db.Model(&User{}).Where("email = ?", commitEmail).Count()
		if err != nil {
			t.Fatalf("Count after commit failed: %v", err)
		}
		if commitCount != 1 {
			t.Fatalf("Expected 1 row after commit, got %d", commitCount)
		}

		rollbackEmail := fmt.Sprintf("pg_tx_rollback_%d@example.com", time.Now().UnixNano())
		err = db.Transaction(func(tx *core.Tx) error {
			_, err := tx.Model(&User{}).Insert(&User{
				Name:  "PGTxRollbackUser",
				Email: rollbackEmail,
				Age:   41,
			})
			if err != nil {
				return err
			}
			return fmt.Errorf("force rollback")
		})
		if err == nil {
			t.Fatalf("Expected error to trigger rollback, got nil")
		}

		rollbackCount, err := db.Model(&User{}).Where("email = ?", rollbackEmail).Count()
		if err != nil {
			t.Fatalf("Count after rollback failed: %v", err)
		}
		if rollbackCount != 0 {
			t.Fatalf("Expected 0 rows after rollback, got %d", rollbackCount)
		}
	})
}
