# AGENTS.md - Guidelines for github.com/shrek82/jorm Code Agents

## Build & Development Commands

```bash
# Build
go build ./...
go build ./core
go build -race ./...

# Testing
go test ./...                                    # All tests
go test -v ./...                                 # Verbose
go test -cover ./...                             # Coverage
go test -race ./...                              # Race detection

# Single Test (most common):
go test -v ./tests -run TestGetModel$           # Exact test
go test -v ./tests -run TestGetModel/Basic$     # Sub-test
go test -v ./core -run TestDBOpen$              # In package
go test -run TestModel                           # Across packages

# Linting & Formatting
go fmt ./...
go vet ./...
golangci-lint run

# Dependencies
go mod tidy
```

## Code Style Guidelines

### Imports
Group into three blocks separated by blank lines: standard library, third-party, project-local (github.com/shrek82/jorm/*). Sort alphabetically.

```go
import (
    "context"
    "database/sql"

    "github.com/mattn/go-sqlite3"

    "github.com/shrek82/jorm/core"
)
```

### Formatting
- Use `gofmt` (tabs, no trailing whitespace)
- Keep lines under 120 characters
- Use Godoc comments for exported types/functions
- No inline comments (e.g., `x = 5 // set to 5`)

### Naming
- Packages: lowercase single word (core, query, model)
- Exported: PascalCase (DB, Query, Find)
- Private: camelCase (buildSQL, parseField)
- Interfaces: end with 'er' for actions (Builder, Executor)

### Type Usage
- Use `any` instead of `interface{}`
- Use pointers for nil-able struct fields
- Use `context.Context` as first param for async ops

### Error Handling
- Always handle errors, never ignore
- Return errors as last return value
- Wrap with context: `fmt.Errorf("operation failed: %w", err)`
- Use `errors.Is()` and `errors.As()` for checking

### Concurrency
- Use `sync.Map` for simple concurrent map access (modelCache, converterCache)
- Use `sync.RWMutex` for complex synchronization (DB health tracking)
- Use `sync.Pool` for buffer reuse (scanBufferPool)
- Lock for minimal scope only

### Function Design
- Small functions (< 50 lines)
- Pointer receivers for mutations, value for read-only
- Return early on errors
- Chain methods return `*Type` for fluent API

### Reflection
- Minimize and cache reflection results (model metadata in sync.Map)
- Use converterCache sync.Map for type conversion caching
- Document reflection-heavy code

### Testing
- Table-driven tests with `t.Run()` sub-tests
- Use `t.Helper()` in test helper functions
- Mock database connections where appropriate
- Test success and failure paths
- Benchmark functions use `setupBenchDB` pattern with cleanup

### Hooks
Implement hook interfaces for lifecycle events:
- `BeforeInserter`, `AfterInserter` - before/after insert
- `BeforeUpdater`, `AfterUpdater` - before/after update
- `BeforeDeleter`, `AfterDeleter` - before/after delete
- `AfterFinder` - after query retrieval

## Architecture Reminders

- **Core**: Connection management, model creation, SQL caching
- **Query**: Chain context, CRUD operations, reflection binding
- **Tx**: Transaction lifecycle, auto-commit/rollback
- **Builder**: SQL construction, dialect handling
- **Model**: Metadata, field mapping, tag parsing
- **Dialect**: Database-specific SQL generation, type mapping
- **Pool**: Connection pooling abstraction (pool.Pool interface)
- **Logger**: Logging interface

Keep concerns separate. Each component has a single responsibility.

## Development Workflow

1. Write tests first (TDD recommended)
2. Implement minimal functionality
3. Run `go fmt ./...`
4. Run `go vet ./...`
5. Run `golangci-lint run`
6. Run `go test ./...`
7. Check coverage
8. Update documentation if API changed

## Project-Specific Patterns

### Package Structure
- **core/**: Main DB, Query, Transaction interfaces
- **model/**: Model metadata, field mapping, tag parsing
- **dialect/**: Database-specific SQL generation
- **logger/**: Logging abstraction
- **pool/**: Connection pooling
- **validator/**: Data validation rules
- **tests/**: Integration and unit tests
- **cmd/**: Code generation tools

### Error Variables
Use predefined error variables from `core/errors.go`:
- `ErrRecordNotFound`, `ErrModelNotFound`, `ErrInvalidModel`
- `ErrInvalidQuery`, `ErrRelationNotFound`, `ErrDuplicateKey`
- `ErrForeignKey`, `ErrConnectionFailed`, `ErrInvalidSQL`

### Model Tags
Use `jorm` tags for field configuration:
- `pk;auto` for primary keys
- `size:100` for field length
- `unique`, `notnull` for constraints
- `fk:Table.Field` for foreign keys
- `auto_time`, `auto_update` for timestamps

### Re-exports
Main `jorm` package re-exports core types and validator rules for convenience.

## Notes

- Target Go version: 1.25.3+
- Use standard library where possible
- Prioritize performance and simplicity
- Use `sync.Map` for model caching
- Maintain backward compatibility
- Follow semantic versioning
