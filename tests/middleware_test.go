package tests

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shrek82/jorm/core"
	"github.com/shrek82/jorm/middleware"
)

func TestMiddleware(t *testing.T) {
	db, err := core.Open("sqlite3", ":memory:", nil)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// 1. Slow Log
	buf := new(bytes.Buffer)
	slowLog := middleware.NewSlowLog(0, "") // Threshold 0 to log everything
	slowLog.SetOutput(buf)
	db.Use(slowLog)

	// Create table
	_, err = db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Insert
	_, err = db.Exec("INSERT INTO users (name) VALUES (?)", "Alice")
	if err != nil {
		t.Fatal(err)
	}

	// Trigger slow query (should log because threshold is 0)
	_, err = db.Table("users").Count()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(buf.String(), "[SLOW SQL]") {
		t.Error("SlowLog should have logged query")
	}

	// 2. File Cache
	cacheDir := "./test_cache"
	os.RemoveAll(cacheDir)
	defer os.RemoveAll(cacheDir)

	fileCache := middleware.NewFileCache(cacheDir, 10*time.Second)
	db.Use(fileCache)

	type User struct {
		ID   int64
		Name string
	}
	var users []User

	// First query (Miss)
	err = db.Table("users").Find(&users)
	if err != nil {
		t.Fatal(err)
	}

	// Check if cache file exists
	entries, _ := os.ReadDir(cacheDir)
	if len(entries) == 0 {
		t.Error("FileCache should have created a cache file")
	}

	// Modify DB directly
	db.Exec("INSERT INTO users (name) VALUES (?)", "Bob")

	// Second query (Hit)
	var users2 []User
	err = db.Table("users").Find(&users2)
	if err != nil {
		t.Fatal(err)
	}

	if len(users2) != 1 {
		t.Errorf("Cache Hit expected 1 user, got %d (Bob should not be visible yet)", len(users2))
	}

	// 3. Circuit Breaker
	// New instance for clean state or just append?
	// DB handles multiple middleware.

	cb := middleware.NewCircuitBreaker(1, 200*time.Millisecond)
	db.Use(cb)

	// Force Error
	_, err = db.Exec("SELECT * FROM non_existent_table")
	if err == nil {
		t.Error("Expected error")
	}

	// Next query should be blocked by CB
	_, err = db.Table("users").Count()
	if err != middleware.ErrCircuitOpen {
		t.Errorf("Expected ErrCircuitOpen, got %v", err)
	}

	// Wait for reset
	time.Sleep(300 * time.Millisecond)
	_, err = db.Table("users").Count()
	if err != nil {
		t.Errorf("Expected success after timeout, got %v", err)
	}

	// 4. Tracing
	tracing := middleware.NewTracing()
	db.Use(tracing)

	ctx := context.WithValue(context.Background(), "request_id", "req-123")
	_, err = db.Table("users").WithContext(ctx).Count()
	if err != nil {
		t.Fatal(err)
	}
}
