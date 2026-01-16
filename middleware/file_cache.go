package middleware

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/shrek82/jorm/core"
)

// FileCacheMiddleware caches query results in the file system.
// To use it, add a duration to the context with key "jorm_cache_ttl".
type FileCacheMiddleware struct {
	CacheDir   string
	DefaultTTL time.Duration
}

func NewFileCache(cacheDir string, defaultTTL ...time.Duration) *FileCacheMiddleware {
	ttl := 5 * time.Minute
	if len(defaultTTL) > 0 {
		ttl = defaultTTL[0]
	}
	return &FileCacheMiddleware{
		CacheDir:   cacheDir,
		DefaultTTL: ttl,
	}
}

func (m *FileCacheMiddleware) Name() string {
	return "FileCache"
}

func (m *FileCacheMiddleware) Init(db *core.DB) error {
	if m.CacheDir == "" {
		return fmt.Errorf("cache directory is required")
	}
	if err := os.MkdirAll(m.CacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	return nil
}

func (m *FileCacheMiddleware) Shutdown() error {
	return nil
}

type fileCacheEntry struct {
	Data      json.RawMessage `json:"data"`
	ExpiresAt time.Time       `json:"expires_at"`
}

func (m *FileCacheMiddleware) Process(ctx context.Context, query *core.Query, next core.QueryFunc) (*core.Result, error) {
	// Check if caching is enabled for this query
	ttl := m.DefaultTTL
	ttlVal := ctx.Value("jorm_cache_ttl")
	
	shouldCache := false
	if ttlVal != nil {
		if t, ok := ttlVal.(time.Duration); ok {
			if t < 0 {
				// Cache() called without args -> use default/permanent
				// For file cache, we use DefaultTTL if set, or a very long duration
				if m.DefaultTTL > 0 {
					ttl = m.DefaultTTL
				} else {
					ttl = 24 * 365 * 100 * time.Hour // "Permanent"
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
	hash := md5.Sum([]byte(key))
	filename := filepath.Join(m.CacheDir, hex.EncodeToString(hash[:])+".json")

	// Try to get from cache
	if data, err := ioutil.ReadFile(filename); err == nil {
		var entry fileCacheEntry
		if err := json.Unmarshal(data, &entry); err == nil {
			if time.Now().Before(entry.ExpiresAt) {
				if query.Dest != nil {
					if err := json.Unmarshal(entry.Data, query.Dest); err == nil {
						return &core.Result{
							Data:         query.Dest,
							RowsAffected: 0,
						}, nil
					}
				}
			} else {
				// Expired, remove file
				_ = os.Remove(filename)
			}
		}
	}

	// Cache miss or failure
	res, err := next(ctx, query)
	if err != nil {
		return res, err
	}

	// Cache the result
	if res.Data != nil {
		dataBytes, err := json.Marshal(res.Data)
		if err == nil {
			entry := fileCacheEntry{
				Data:      dataBytes,
				ExpiresAt: time.Now().Add(ttl),
			}
			fileBytes, _ := json.Marshal(entry)
			_ = ioutil.WriteFile(filename, fileBytes, 0644)
		}
	}

	return res, nil
}
