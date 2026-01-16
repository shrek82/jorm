package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shrek82/jorm/core"
)

// RedisCacheMiddleware caches query results in Redis.
// To use it, add a duration to the context with key "jorm_cache_ttl".
type RedisCacheMiddleware struct {
	Client *redis.Client
}

func NewRedisCache(opt *redis.Options) *RedisCacheMiddleware {
	return &RedisCacheMiddleware{
		Client: redis.NewClient(opt),
	}
}

func (m *RedisCacheMiddleware) Name() string {
	return "RedisCache"
}

func (m *RedisCacheMiddleware) Init(db *core.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.Client.Ping(ctx).Err()
}

func (m *RedisCacheMiddleware) Shutdown() error {
	return m.Client.Close()
}

func (m *RedisCacheMiddleware) Process(ctx context.Context, query *core.Query, next core.QueryFunc) (*core.Result, error) {
	// Check if caching is enabled for this query
	ttlVal := ctx.Value("jorm_cache_ttl")
	if ttlVal == nil {
		return next(ctx, query)
	}

	var ttl time.Duration
	if t, ok := ttlVal.(time.Duration); ok {
		if t == 0 {
			// Cache(0) -> disable cache
			return next(ctx, query)
		}
		if t < 0 {
			// Cache() -> permanent (Redis uses 0 for no expiration)
			ttl = 0
		} else {
			ttl = t
		}
	} else {
		return next(ctx, query)
	}

	// Generate cache key
	sqlStr, args := query.GetSelectSQL()
	if sqlStr == "" {
		// Not a select query or SQL not ready?
		// Try to build it if possible, but middleware is usually executed before query execution
		// core.Query.GetSelectSQL() builds it from builder if rawSQL is empty.
	}

	// Create a simple cache key
	// In a real app, might want to hash this
	key := fmt.Sprintf("jorm:cache:%s:%v", sqlStr, args)

	// Try to get from cache
	val, err := m.Client.Get(ctx, key).Result()
	if err == nil {
		// Cache hit
		if query.Dest != nil {
			// Unmarshal into a temporary object to avoid corrupting Dest on failure
			destType := reflect.TypeOf(query.Dest)
			if destType.Kind() == reflect.Ptr {
				temp := reflect.New(destType.Elem()).Interface()
				if err := json.Unmarshal([]byte(val), temp); err != nil {
					// Failed to unmarshal, proceed to DB
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
			m.Client.Set(ctx, key, data, ttl)
		}
	}

	return res, nil
}
