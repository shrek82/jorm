package tests

import (
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shrek82/jorm/core"
	"github.com/shrek82/jorm/middleware"
)

func TestMultiLevelCache(t *testing.T) {
	dbFile := "./test_multilevel.db"
	cacheDir := "./cache_multilevel"

	// Cleanup
	os.Remove(dbFile)
	os.RemoveAll(cacheDir)
	defer os.Remove(dbFile)
	defer os.RemoveAll(cacheDir)

	// 1. Initialize DB and Data
	// We use a separate connection for setup to avoid cache pollution
	setupDB, err := core.Open("sqlite3", dbFile, nil)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	_, err = setupDB.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatal(err)
	}
	_, err = setupDB.Exec("INSERT INTO users (name) VALUES (?)", "Alice")
	if err != nil {
		t.Fatal(err)
	}
	setupDB.Close()

	// 2. Scenario 1: Cold Start -> Mem & File Backfill
	// DB1 has MemCache1 and FileCache
	db1, err := core.Open("sqlite3", dbFile, nil)
	if err != nil {
		t.Fatal(err)
	}
	mem1 := middleware.NewMemoryCache()
	fileCache := middleware.NewFileCache(cacheDir) // Shared file cache config
	db1.Use(mem1, fileCache)

	type User struct {
		ID   int64
		Name string
	}

	var users1 []User
	// Query 1: Miss Mem -> Miss File -> Hit DB -> Backfill File -> Backfill Mem
	err = db1.Table("users").Cache().Find(&users1)
	if err != nil {
		t.Fatal(err)
	}
	if len(users1) != 1 || users1[0].Name != "Alice" {
		t.Errorf("Expected Alice, got %v", users1)
	}

	// Verify File Cache exists
	entries, _ := os.ReadDir(cacheDir)
	if len(entries) == 0 {
		t.Error("File cache should be created")
	}

	// 3. Scenario 2: Memory Hit
	// Modify DB directly (using a raw connection to bypass cache invalidation if any)
	rawDB, _ := core.Open("sqlite3", dbFile, nil)
	rawDB.Exec("UPDATE users SET name = ? WHERE id = ?", "Bob", 1)
	rawDB.Close()

	var users2 []User
	// Query 2: Hit Mem (Should still be Alice)
	err = db1.Table("users").Cache().Find(&users2)
	if err != nil {
		t.Fatal(err)
	}
	if len(users2) != 1 || users2[0].Name != "Alice" {
		t.Errorf("Expected hit Memory (Alice), got %v", users2)
	}

	db1.Close() // Close db1 to simulate app restart / new instance

	// 4. Scenario 3: File Hit -> Mem Backfill
	// DB2 has NEW MemCache2 but SAME FileCache
	db2, err := core.Open("sqlite3", dbFile, nil)
	if err != nil {
		t.Fatal(err)
	}
	mem2 := middleware.NewMemoryCache()
	fileCache2 := middleware.NewFileCache(cacheDir)
	db2.Use(mem2, fileCache2)

	var users3 []User
	// Query 3: Miss Mem2 -> Hit FileCache (Alice) -> Backfill Mem2
	// DB has "Bob", so if we hit DB we get Bob. If we hit File, we get Alice.
	err = db2.Table("users").Cache().Find(&users3)
	if err != nil {
		t.Fatal(err)
	}
	if len(users3) != 1 || users3[0].Name != "Alice" {
		t.Errorf("Expected hit File (Alice), got %v. Note: DB has Bob.", users3)
	}

	// 5. Scenario 4: Mem2 Hit (verified backfill)
	// Clear File Cache manually to prove we are hitting Mem2 now
	os.RemoveAll(cacheDir)

	var users4 []User
	// Query 4: Hit Mem2 (Alice)
	err = db2.Table("users").Cache().Find(&users4)
	if err != nil {
		t.Fatal(err)
	}
	if len(users4) != 1 || users4[0].Name != "Alice" {
		t.Errorf("Expected hit Memory2 (Alice) after backfill, got %v", users4)
	}

	db2.Close()
}
