package tests

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shrek82/jorm/core"
	"github.com/shrek82/jorm/middleware"
)

func TestConcurrencyAndRobustness(t *testing.T) {
	// Setup DB
	// Use shared cache for in-memory DB to support connection pooling
	opts := &core.Options{
		MaxOpenConns: 1, // Force single connection to avoid lock issues with sqlite shared cache if any
	}
	db, err := core.Open("sqlite3", "file::memory:?cache=shared", opts)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	defer db.Close()

	// Setup Middlewares
	// 1. SlowLog (write to discard to avoid spamming output)
	slowLog := middleware.NewSlowLog(10*time.Millisecond, "")
	// We can't set output to io.Discard easily without the exposed method or package variable,
	// but default is stdout. We'll leave it or set to a temp file.
	// Actually we added SetOutput.
	slowLog.SetOutput(os.Stdout)
	db.Use(slowLog)

	// 2. Circuit Breaker
	cb := middleware.NewCircuitBreaker(10, 100*time.Millisecond)
	db.Use(cb)

	// 3. File Cache
	cacheDir := "./bench_cache"
	os.RemoveAll(cacheDir)
	defer os.RemoveAll(cacheDir)
	fileCache := middleware.NewFileCache(cacheDir, 5*time.Second)
	db.Use(fileCache)

	// Create Table
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS products (id INTEGER PRIMARY KEY, name TEXT, price INTEGER)")
	if err != nil {
		t.Fatal(err)
	}

	// Insert Data
	for i := 0; i < 10; i++ {
		_, err = db.Exec("INSERT INTO products (name, price) VALUES (?, ?)", fmt.Sprintf("Product%d", i), i*10)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Concurrency Test
	var wg sync.WaitGroup
	workers := 20
	iterations := 50

	errCh := make(chan error, workers*iterations)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Mix of operations

				// 1. Find with Cache
				var products []struct {
					ID    int
					Name  string
					Price int
				}
				err := db.Table("products").Cache().Limit(5).Find(&products)
				if err != nil {
					errCh <- fmt.Errorf("worker %d find failed: %w", id, err)
					return
				}
				if len(products) != 5 {
					errCh <- fmt.Errorf("worker %d expected 5 products, got %d", id, len(products))
					return
				}

				// 2. Count with Cache
				count, err := db.Table("products").Cache().Count()
				if err != nil {
					errCh <- fmt.Errorf("worker %d count failed: %w", id, err)
					return
				}
				if count != 10 {
					errCh <- fmt.Errorf("worker %d expected count 10, got %d", id, count)
					return
				}

				// 3. Trigger Circuit Breaker (occasionally)
				if id == 0 && j == 0 {
					// Force error to check CB state safety (not checking logic, just race freedom)
					db.Exec("SELECT * FROM non_existent")
				}
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatal(err)
	}
}
