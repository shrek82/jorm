package tests

import (
	"fmt"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shrek82/jorm/core"
)

type BenchUser struct {
	ID    int64  `jorm:"primary_key;auto_increment"`
	Name  string `jorm:"column:name"`
	Email string `jorm:"column:email"`
	Age   int    `jorm:"column:age"`
}

func (BenchUser) TableName() string {
	return "bench_user"
}

func setupBenchDB(b *testing.B) (*core.DB, func()) {
	dbPath := "bench.db"
	// Ensure clean start
	os.Remove(dbPath)

	db, err := core.Open("sqlite3", dbPath, nil)
	if err != nil {
		b.Fatalf("Failed to open DB: %v", err)
	}

	// Disable logging for benchmark to measure pure DB performance
	db.SetLogger(nil)

	err = db.AutoMigrate(&BenchUser{})
	if err != nil {
		b.Fatalf("Failed to migrate: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}

	return db, cleanup
}

func BenchmarkInsert(b *testing.B) {
	db, cleanup := setupBenchDB(b)
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		user := &BenchUser{
			Name:  fmt.Sprintf("User%d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
			Age:   20,
		}
		_, err := db.Model(user).Insert(user)
		if err != nil {
			b.Fatalf("Insert failed at %d: %v", i, err)
		}
	}
}

func BenchmarkBatchInsert100(b *testing.B) {
	db, cleanup := setupBenchDB(b)
	defer cleanup()

	batchSize := 100
	users := make([]*BenchUser, batchSize)
	for i := 0; i < batchSize; i++ {
		users[i] = &BenchUser{
			Name:  fmt.Sprintf("User%d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
			Age:   20,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.Model(&BenchUser{}).BatchInsert(users)
		if err != nil {
			b.Fatalf("BatchInsert failed: %v", err)
		}
	}
	b.SetBytes(int64(batchSize))
}

func BenchmarkBatchInsert1000(b *testing.B) {
	db, cleanup := setupBenchDB(b)
	defer cleanup()

	batchSize := 1000
	users := make([]*BenchUser, batchSize)
	for i := 0; i < batchSize; i++ {
		users[i] = &BenchUser{
			Name:  fmt.Sprintf("User%d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
			Age:   20,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.Model(&BenchUser{}).BatchInsert(users)
		if err != nil {
			b.Fatalf("BatchInsert failed: %v", err)
		}
	}
	b.SetBytes(int64(batchSize))
}
