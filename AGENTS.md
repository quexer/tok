# AGENTS.md

This guide helps agentic coding assistants understand and work with this codebase effectively.

## Build, Lint, and Test Commands

```bash
# Build the project
make build

# Format and tidy code
make fmt

# Generate mocks
make mock

# Run all tests
make test

# Run a single test file
ginkgo -r <test_file_name>

# Run a specific test spec
ginkgo -r -focus="<Describe>/<It>"
```

## Code Style Guidelines

### Project Overview
This is a Go library ("tok" - talk) for creating IM applications with support for TCP and WebSocket connections, featuring a pluggable ConAdapter interface for custom connection types.

### Import Organization
1. Standard library imports first
2. Third-party imports second
3. Use explicit imports (no periods) unless dot imports are clearly justified (e.g., testing frameworks)
4. Example pattern:
```go
import (
    "context"
    "errors"
    "log/slog"
    "time"

    "github.com/coder/websocket"
    "github.com/gorilla/websocket"
)
```

### Formatting
- Use `go fmt ./...` to format code
- Run `go mod tidy` after adding dependencies
- Use `make fmt` to run both

### Naming Conventions

#### Types and Interfaces
- Interfaces: PascalCase (Actor, BeforeReceiveHandler, Queue, ConAdapter)
- Public structs: PascalCase (Hub, Device, WsHandler, HubConfig)
- Private structs: lowerCamelCase (connection, tcpAdapter, downFrame)

#### Functions and Methods
- Public functions: PascalCase (CreateDevice, Listen, CreateWsHandler)
- Private functions: lowerCamelCase (createHub, connExclude, beforeSend)
- Public methods: PascalCase (UID, ID, GetMeta, PutMeta, CheckOnline)
- Private methods: lowerCamelCase (readLoop, triggerOffline, goOnline)

#### Variables and Constants
- Constants: UPPER_SNAKE_CASE (TCPMaxPackLen, tcpHeaderLen)
- Package-level variables: lowerCamelCase (expOnline, expUp, expDown)
- Local variables: lowerCamelCase (hub, conn, err, data)
- Function parameters: lowerCamelCase (uid, to, b, ttl)
- Receiver names: short, context-aware (p for Hub, conn for connection)
- Bool parameters: descriptive names with negation avoided when possible (online, isClosed)

#### Error Variables
- Predefined errors: ErrCamelCase (ErrOffline, ErrQueueRequired, ErrCacheFailed)

### Type Conventions
- Use `interface{}` for user IDs (allows flexibility)
- Use `[]byte` for message payloads
- Use `context.Context` for all operations that may need cancellation
- Use `chan` for communication between goroutines

### Error Handling
- Define package-level error variables for common errors
- Use `fmt.Errorf` with `%w` for wrapping errors
- Use `errors.Is()` for error checking
- Return errors, don't panic (except in package initialization/fatal conditions)
- Log errors with `slog` (log/slog package) using structured logging:
  - `slog.Warn("message", "err", err)`
  - `slog.Error("message", "err", err)`
  - `slog.Debug("message", "key", value)`

### Configuration Pattern
Use functional options pattern for configuration:
```go
type HubConfigOption func(*HubConfig)

func WithHubConfigQueue(q Queue) HubConfigOption {
    return func(hc *HubConfig) {
        hc.q = q
    }
}
```

### Mock Generation
- Use `//go:generate mockgen` comments for mock generation
- Mocks are generated in `mocks/` directory
- Run `make mock` to generate mocks
- Use `go.uber.org/mock/gomock` for testing

### Thread Safety
- Use `sync.Mutex` for write operations (e.g., connection.Write)
- Use `atomic` operations for simple flags and counters
- Document thread-safety guarantees in struct comments
- Use `sync.Map` for concurrent map access when appropriate

### Testing
- Use Ginkgo v2 + Gomega for BDD-style tests
- Test files named `<file>_test.go`
- Use `_test` package suffix for test files
- Use `BeforeEach`, `JustBeforeEach`, `AfterEach` for test lifecycle
- Use `Describe`, `It`, `Context` for test organization
- Use `gomock.Controller` for mock management in BeforeEach

### Interface Design
- Prefer small, focused interfaces
- Use adapter pattern for different connection types (ConAdapter)
- Implement optional handlers as separate interfaces
- Use dependency injection via configuration

### Code Organization
- Each file has a single responsibility
- Use `//go:generate` comments at top of files with interfaces to mock
- Document public interfaces and types with comments
- Example file structure: hub.go (logic), hub_config.go (configuration), tcp_conn.go (TCP adapter)

### Concurrency
- Use channels for communication between goroutines
- Prefer non-blocking channel sends with select statements
- Use goroutines for I/O operations and parallel processing
- Ensure proper cleanup with `defer` and context cancellation
- Use atomic operations for exactly-once semantics (e.g., triggerOffline)

### Logging
- Use `log/slog` for structured logging
- Use appropriate log levels: Debug, Info, Warn, Error
- Include contextual information as key-value pairs
- Prefix package-specific logs with package name in brackets: "[tok]"

### Documentation
- Use Go doc comments for exported types, functions, and methods
- Include parameter descriptions and return value explanations
- Document thread-safety guarantees
- Example:
```go
// Send message to someone.
// ttl is expiry seconds. 0 means only send to online user
// If ttl = 0 and user is offline, ErrOffline will be returned.
func (p *Hub) Send(ctx context.Context, to interface{}, b []byte, ttl uint32) error
```

### Files and Directories
- `tok.go` - Entry and core types
- `hub.go` - Hub logic for connection management and message dispatch
- `hub_config.go` - Configuration and options
- `conn.go` - Connection wrapper and ConAdapter interface
- `tcp_conn.go` - TCP adapter
- `ws_conn.go`, `ws_gorilla.go`, `ws_x.go`, `ws_coder.go` - WebSocket adapters
- `device.go` - Device abstraction
- `q.go` - Queue interface
- `memory_q.go` - In-memory queue implementation
- `mocks/` - Generated mocks
- `example/` - Example implementations

### Before Committing
Always run:
```bash
make fmt
make mock
make test
```

No additional lint/typecheck commands are configured in this project.
