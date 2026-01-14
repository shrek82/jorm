# AGENTS.md - Guidelines for github.com/shrek82/jorm Code Agents

## Build & Development Commands

```bash
# Build
go build ./...
go build ./core
go build -race ./...

# Testing
go test ./...
go test -v ./...                    # Verbose
go test -cover ./...                 # With coverage
go test -race ./...                  # Race detector
go test -v ./core -run TestDBOpen   # Specific test file
go test -v ./core -run TestDBOpen/Success  # Specific test function
go test -v ./query/...              # Specific package
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

# Linting
go fmt ./...
go vet ./...
golangci-lint run
go fmt ./... && go vet ./... && golangci-lint run

# Dependencies
go mod tidy
go mod graph
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
- No inline comments

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
- Use `sync.Map` for simple concurrent map access (as per architecture)
- Use `sync.RWMutex` for complex synchronization
- Lock for minimal scope only

### Function Design
- Small functions (< 50 lines)
- Pointer receivers for mutations, value for read-only
- Return early on errors
- Chain methods return `*Type` for fluent API

### Reflection
- Minimize and cache reflection results (model metadata)
- Document reflection-heavy code

### Testing
- Table-driven tests with `t.Run()` sub-tests
- Mock database connections where appropriate
- Test success and failure paths

## Architecture Reminders

- **Core**: Connection management, model creation, SQL caching
- **Query**: Chain context, CRUD operations, reflection binding
- **Tx**: Transaction lifecycle, auto-commit/rollback
- **Builder**: SQL construction, dialect handling
- **Model**: Metadata, field mapping, tag parsing
- **Dialect**: Database-specific SQL generation, type mapping
- **Pool**: Connection pooling abstraction
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

## Notes

- Target Go version: 1.25.3+
- Use standard library where possible
- Prioritize performance and simplicity
- Use `sync.Map` for model caching
- Maintain backward compatibility
- Follow semantic versioning
