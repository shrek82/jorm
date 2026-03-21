# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**jorm** is a lightweight, high-performance Go ORM library supporting MySQL, PostgreSQL, SQLite, Oracle, and SQL Server. It provides chainable query operations, transaction management, hooks, connection pooling, auto-migration, relation preloading, and data validation.

## Build & Test Commands

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./tests/
go test ./core/
go test ./model/

# Run a single test (requires database DSN)
MYSQL_TEST_DSN="user:pass@tcp(127.0.0.1:3306)/db?parseTime=true" go test -run TestMySQLIntegration ./tests/

# Run benchmarks
go test -bench=. ./tests/

# Build
go build ./...
```

## Architecture Overview

### Package Structure

| Package | Purpose |
|---------|---------|
| `core` | Central ORM engine: `DB`, `Query`, `Builder`, connection pooling, health checks |
| `model` | Reflection-based model metadata extraction and hook interfaces |
| `dialect` | Database-specific SQL generation (MySQL, PostgreSQL, SQLite, Oracle, SQL Server) |
| `middleware` | Query interceptors: caching (memory/redis/file), slow log, circuit breaker, tracing |
| `validator` | Data validation rules and validation execution |
| `pool` | Database connection pool abstraction |
| `logger` | Structured logging for SQL and errors |
| `tests` | Integration and unit tests |

### Key Components

**`core.DB`** (`core/db.go:35`): Central orchestrator managing connection pool, dialect, logger, middleware chain, and health tracking with cooldown logic for connection failures.

**`core.Query`** (`core/query.go:27`): Fluent query builder and executor. Supports chainable methods (Where, OrderBy, Limit, etc.), preloads, and executes CRUD via middleware pipeline.

**`core.Builder`** (`core/builder.go:15`): SQL statement constructor. Pools builders via `sync.Pool`, handles dialect-specific placeholder conversion (`?` → `$1` for PostgreSQL).

**`model.Model`** (`model/model.go`): Reflection-based metadata cache. Extracts struct fields, tags, primary keys, and hook capabilities. Cached via `sync.Map`.

### Execution Flow

1. User calls `db.Model(&User{}).Where("id=?", 1).First(&user)`
2. `Query` builds SQL via `Builder` → dialect-aware SQL generation
3. Middleware pipeline processes query (cache → slow log → circuit breaker → exec)
4. Results scanned via `scanPlan` (cached reflection plan) into destination
5. Hooks executed (Before/After Find, Insert, Update, Delete)
6. Preloads resolved if configured

### Hook Interfaces (`model/hooks.go`)

```go
BeforeFinder/AfterFinder, BeforeInserter/AfterInserter,
BeforeUpdater/AfterUpdater, BeforeDeleter/AfterDeleter
```

Models implement these interfaces to hook into ORM lifecycle events.

### Middleware Pattern (`core/middleware.go`)

```go
type QueryMiddleware interface {
    Name() string
    Init(*DB) error
    Process(ctx context.Context, query *Query, next QueryFunc) (*Result, error)
}
```

Middleware wraps query execution. Order matters: first registered = outermost wrapper.

## Design Patterns

- **Pool Reuse**: `Builder`, `scanBuffer` use `sync.Pool` to reduce allocations
- **Reflection Caching**: Model metadata, scan plans, converters cached in `sync.Map`
- **Health Cooldown**: Connection errors trigger 5-second cooldown before retrying
- **Dialect Abstraction**: All SQL generation delegated to dialect-specific implementations

## Common Patterns

**Transaction with rollback on error:**
```go
err := db.Transaction(func(tx *core.Tx) error {
    _, err := tx.Model(&user).Insert(user)
    return err // non-nil triggers automatic rollback
})
```

**Custom middleware:**
```go
db.Use(NewMemoryCache(5*time.Minute))
```

**Pagination:**
```go
page, err := db.Model(&User{}).Paginate(pageNum, pageSize, &users)
```

**Preload relations:**
```go
db.Model(&User{}).Preload("Posts").Find(&users)
```
