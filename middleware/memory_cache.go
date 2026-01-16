package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/shrek82/jorm/core"
)

// MemoryCacheMiddleware caches query results in memory.
// To use it, add a duration to the context with key "jorm_cache_ttl".
type MemoryCacheMiddleware struct {
	items      map[string]memoryCacheEntry
	mu         sync.RWMutex
	stopClean  chan struct{}
	DefaultTTL time.Duration
}

type memoryCacheEntry struct {
	Data      []byte
	ExpiresAt time.Time
}

func NewMemoryCache(defaultTTL ...time.Duration) *MemoryCacheMiddleware {
	ttl := 5 * time.Minute
	if len(defaultTTL) > 0 {
		ttl = defaultTTL[0]
	}
	return &MemoryCacheMiddleware{
		items:      make(map[string]memoryCacheEntry),
		stopClean:  make(chan struct{}),
		DefaultTTL: ttl,
	}
}

func (m *MemoryCacheMiddleware) Name() string {
	return "MemoryCache"
}

func (m *MemoryCacheMiddleware) Init(db *core.DB) error {
	// Start cleanup goroutine
	go m.cleanupLoop()
	return nil
}

func (m *MemoryCacheMiddleware) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopClean:
			return
		case <-ticker.C:
			m.cleanup()
		}
	}
}

func (m *MemoryCacheMiddleware) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for k, v := range m.items {
		if !v.ExpiresAt.IsZero() && now.After(v.ExpiresAt) {
			delete(m.items, k)
		}
	}
}

func (m *MemoryCacheMiddleware) Shutdown() error {
	close(m.stopClean)
	return nil
}

func (m *MemoryCacheMiddleware) Process(ctx context.Context, query *core.Query, next core.QueryFunc) (*core.Result, error) {
	// Check if caching is enabled for this query
	ttl := m.DefaultTTL
	ttlVal := ctx.Value("jorm_cache_ttl")

	shouldCache := false
	if ttlVal != nil {
		if t, ok := ttlVal.(time.Duration); ok {
			if t == 0 {
				// Cache(0) -> disable cache
				return next(ctx, query)
			} else if t == -1 {
				// Cache(-1) -> Permanent
				ttl = 24 * 365 * 100 * time.Hour // "Permanent"
				shouldCache = true
			} else if t == -2 {
				// Cache() -> use default if set, else 24h
				if m.DefaultTTL > 0 {
					ttl = m.DefaultTTL
				} else {
					ttl = 24 * time.Hour // Default fallback
				}
				shouldCache = true
			} else if t > 0 {
				ttl = t
				shouldCache = true
			}
		}
	}

	if !shouldCache {
		return next(ctx, query)
	}

	// Generate cache key
	sqlStr, args := query.GetSelectSQL()
	key := fmt.Sprintf("jorm:cache:%s:%v", sqlStr, args)

	// Try to get from cache
	m.mu.RLock()
	entry, found := m.items[key]
	m.mu.RUnlock()

	if found {
		if entry.ExpiresAt.IsZero() || time.Now().Before(entry.ExpiresAt) {
			if query.Dest != nil {
				// Unmarshal into a temporary object to avoid corrupting Dest on failure
				destType := reflect.TypeOf(query.Dest)
				if destType.Kind() == reflect.Ptr {
					temp := reflect.New(destType.Elem()).Interface()
					if err := json.Unmarshal(entry.Data, temp); err != nil {
						// Failed to unmarshal, ignore cache
					} else {
						// Success, copy to Dest
						reflect.ValueOf(query.Dest).Elem().Set(reflect.ValueOf(temp).Elem())
						return &core.Result{
							Data:         query.Dest,
							RowsAffected: 0,
						}, nil
					}
				}
			}
		} else {
			// Expired, delete (lazy delete)
			m.mu.Lock()
			delete(m.items, key)
			m.mu.Unlock()
		}
	}

	// Cache miss or failure
	res, err := next(ctx, query)
	if err != nil {
		return res, err
	}

	// Cache the result
	if res.Data != nil {
		data, err := json.Marshal(res.Data)
		if err == nil {
			m.mu.Lock()
			m.items[key] = memoryCacheEntry{
				Data:      data,
				ExpiresAt: time.Now().Add(ttl),
			}
			m.mu.Unlock()
		}
	}

	return res, nil
}
