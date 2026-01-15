package tests

import (
	"fmt"
	"sync"
	"testing"

	"github.com/shrek82/jorm/model"
)

type ConcurrentUser struct {
	ID    int64 `jorm:"pk auto"`
	Name  string
	Posts []ConcurrentPost `jorm:"relation:has_many"`
}

type ConcurrentPost struct {
	ID     int64 `jorm:"pk auto"`
	Title  string
	UserID int64
}

func TestRelationCacheConcurrency(t *testing.T) {
	const goroutines = 20
	const iterations = 50

	var wg sync.WaitGroup
	errs := make(chan error, goroutines*iterations)

	m, err := model.GetModel(&ConcurrentUser{})
	if err != nil {
		t.Fatalf("Failed to get initial model: %v", err)
	}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Trigger cache invalidation periodically
				if j%10 == 0 {
					model.InvalidateRelationCache()
				}

				rel, err := model.GetRelation(m, "Posts")
				if err != nil {
					errs <- fmt.Errorf("goroutine %d iteration %d failed: %v", id, j, err)
					return
				}
				if rel == nil || rel.Name != "Posts" {
					errs <- fmt.Errorf("goroutine %d iteration %d got invalid relation", id, j)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}
