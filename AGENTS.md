# AGENTS.md - Guidelines for JORM3 Code Agents

This file contains essential information for agentic coding tools working on the JORM3 ORM library.

---

## Build & Development Commands

### Building
```bash
# Build all packages
go build ./...

# Build specific package
go build ./core

# Build with race detector
go build -race ./...

# Build for all platforms
gox -os="linux darwin windows" -arch="amd64 arm64" ./...
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detector
go test -race ./...

# Run a specific test file
go test -v ./core -run TestDBOpen

# Run a specific test function
go test -v ./core -run TestDBOpen/Success

# Run tests for a specific package
go test -v ./query/...

# Run tests with verbose output
go test -v ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Linting & Formatting
```bash
# Format code
go fmt ./...

# Run go vet
go vet ./...

# Run golangci-lint (recommended)
golangci-lint run

# Format and lint in one command
go fmt ./... && go vet ./... && golangci-lint run

# Check for common issues
staticcheck ./...
```

### Other Useful Commands
```bash
# List all packages
go list ./...

# View dependencies
go mod tidy
go mod graph

# Update dependencies
go get -u ./...

# View module information
go mod why <package>
```

---

## Code Style Guidelines

### Imports
- Group imports into three blocks separated by blank lines:
  1. Standard library
  2. Third-party packages
  3. Project-local packages (jorm3/*)
- Sort imports alphabetically within each group
- Use blank line between groups
- Avoid unused imports (use `goimports` or IDE auto-import)

Example:
```go
import (
    "context"
    "database/sql"
    "reflect"
    "sync"

    "github.com/go-sql-driver/mysql"

    "jorm3/core"
    "jorm3/dialect"
)
```

### Formatting
- Use `gofmt` or `go fmt` for all code (enforced)
- Use tabs for indentation (Go standard)
- Keep lines under 120 characters when practical
- Use blank lines to separate logical sections
- No trailing whitespace

### Naming Conventions
- **Package names**: lowercase, single word, short, descriptive (e.g., `core`, `query`, `model`)
- **Exported types/functions**: PascalCase (e.g., `DB`, `Query`, `Find`)
- **Private types/functions**: camelCase (e.g., `buildSQL`, `parseField`)
- **Interfaces**: typically end with `er` if action-based (e.g., `Builder`, `Executor`, `Logger`)
- **Constants**: PascalCase for exported, camelCase for private
- **Struct fields**: PascalCase for exported, camelCase for private

Example:
```go
// Exported
type DB struct { ... }
func (db *DB) Open() error { ... }

// Private
type dbConfig struct { ... }
func (db *dbConfig) parse() error { ... }
```

### Type Usage & Types
- Prefer concrete types over interface{} when type is known
- Use `any` instead of `interface{}` for Go 1.18+
- Use pointers for nil-able struct fields in models
- Use value types for small structs unless nil is meaningful
- Leverage generics where appropriate (Go 1.18+ features)
- Use `context.Context` as first parameter for async/time-sensitive operations

Example:
```go
// Good
func (q *Query) Where(condition string, args ...any) *Query

// Avoid - prefer specific types
func Process(data interface{}) error

// Good - use any
func Process(data any) error
```

### Error Handling
- Always handle errors, never ignore them
- Return errors as last return value
- Use error wrapping for context: `fmt.Errorf("operation failed: %w", err)`
- Define custom error types in `errors.go` or similar
- Use `errors.Is()` and `errors.As()` for error checking
- Return sentinel errors from `errors` package for comparison

Example:
```go
// Custom error type
type JormError struct {
    Code    int
    Message string
    Err     error
}

func (e *JormError) Error() string {
    return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e *JormError) Unwrap() error { return e.Err }

// Error handling
result, err := db.Exec(query)
if err != nil {
    return fmt.Errorf("failed to execute query: %w", err)
}

// Check error type
if errors.Is(err, ErrRecordNotFound) {
    // handle not found
}
```

### Concurrency
- Use `sync.Map` for simple concurrent map access (as per architecture)
- Use `sync.RWMutex` for more complex synchronization needs
- Use channels for communication between goroutines
- Avoid sharing state; prefer passing values
- Always lock for the minimal scope possible

Example:
```go
// Using sync.Map for caching (as per architecture)
var modelCache sync.Map

func getModel(key string) (*Model, bool) {
    if val, ok := modelCache.Load(key); ok {
        return val.(*Model), true
    }
    return nil, false
}
```

### Function Design
- Prefer small, focused functions (< 50 lines when possible)
- Use receiver functions for methods on types
- Value receivers for immutable operations, pointer receivers for mutations
- Return early on errors to reduce nesting
- Accept interfaces, return concrete types

Example:
```go
// Good - pointer receiver for mutation
func (q *Query) Where(condition string, args ...any) *Query {
    q.clauses = append(q.clauses, &Clause{Type: WhereClause, Value: condition})
    q.args = append(q.args, args...)
    return q
}

// Good - value receiver for read-only
func (m *Model) TableName() string {
    return m.tableName
}
```

### Reflection Usage
- Minimize reflection usage (performance impact)
- Cache reflection results (e.g., model metadata)
- Use reflection only for dynamic type handling where unavoidable
- Document reflection-heavy code clearly
- Consider code generation for high-performance paths

### Documentation
- Exported functions must have godoc comments
- Use complete sentences for godoc comments
- Include examples in godoc for complex APIs
- Comment non-obvious logic
- Use TODO/FIXME comments sparingly with context

Example:
```go
// Model creates a new Query for the given model.
// The model parameter should be a pointer to a struct or a struct value.
// Example:
//
//   var user User
//   query := db.Model(&user)
func (db *DB) Model(value any) *Query {
    // ...
}
```

### Project-Specific Conventions

#### Chain Methods
- All chain methods return `*Type` for fluent API
- Always return the same object (don't return copies)
- Use immutable where possible, mutable when performance-critical

#### Builder Pattern
- Build SQL incrementally through clauses
- Support both fluent and direct construction
- Use `Build()` method to generate final SQL and args

#### Context Support
- Accept `context.Context` as first parameter for async operations
- Provide `*Context()` variant of methods that accept context
- Pass context through the call chain

Example:
```go
// WithContext sets the context for this query
func (q *Query) WithContext(ctx context.Context) *Query {
    q.ctx = ctx
    return q
}

// Direct context method
func (q *Query) FindContext(ctx context.Context, dest any) error {
    // ...
}
```

### Testing Conventions
- Table-driven tests for multiple scenarios
- Use `t.Run()` for sub-tests
- Mock database connections where appropriate
- Test both success and failure paths
- Include integration tests for real database operations

Example:
```go
func TestDBOpen(t *testing.T) {
    tests := []struct {
        name    string
        dsn     string
        wantErr bool
    }{
        {"valid connection", "mock://valid", false},
        {"invalid dsn", "invalid://dsn", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            db, err := Open(tt.dsn, nil)
            if (err != nil) != tt.wantErr {
                t.Errorf("Open() error = %v, wantErr %v", err, tt.wantErr)
            }
            if db != nil {
                db.Close()
            }
        })
    }
}
```

### Performance Considerations
- Use connection pooling (as per architecture)
- Cache prepared statements
- Use batch operations for bulk inserts/updates
- Avoid N+1 queries
- Profile with `go test -bench` and `pprof`

---

## Architecture Reminders

- **Core responsibilities**: Connection management, model creation, SQL caching
- **Query**: Chain context, CRUD operations, reflection-based binding
- **Tx**: Transaction lifecycle, auto-commit/rollback
- **Builder**: SQL construction, dialect handling
- **Model**: Metadata, field mapping, tag parsing
- **Dialect**: Database-specific SQL generation, type mapping
- **Pool**: Connection pooling abstraction
- **Logger**: Logging interface and implementations

Keep concerns separate. Each component has a clear, single responsibility.

---

## Development Workflow

1. Write tests first (TDD recommended)
2. Implement minimal functionality
3. Run `go fmt ./...`
4. Run `go vet ./...`
5. Run `golangci-lint run` if available
6. Run `go test ./...`
7. Check coverage with `go test -cover ./...`
8. Update documentation if API changed
9. Commit changes with clear messages

---

## Notes

- Target Go version: 1.25.3+
- Use standard library where possible
- Minimize external dependencies
- Prioritize performance and simplicity
- Maintain backward compatibility when possible
- Follow semantic versioning for releases
