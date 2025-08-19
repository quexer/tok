# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Commands

### Testing
- Run all tests: `go test ./...`
- Run tests with verbose output: `go test -v ./...`
- Run tests with race detection: `go test -race ./...`
- Run a specific test: `go test -run TestNamePattern ./...`
- Run tests without cache: `go test -count=1 ./...`

### Building
- Build the library: `go build ./...`
- Install dependencies: `go mod download`
- Update dependencies: `go mod tidy`

### Development
- Format code: `go fmt ./...`
- Run static analysis: `go vet ./...`
- Run examples: `go run example/[gorilla_server|xws_server]/main.go`

## Architecture Overview

The tok library is an IM (Instant Messaging) framework with a modular design that separates network adapters from core logic:

### Core Components

1. **Hub (`hub.go`)**: Central message dispatcher that manages connections and routes messages between devices. Uses channel-based architecture for concurrent operations.

2. **Actor Interface (`tok.go`)**: Primary interface requiring only `OnReceive(dv *Device, data []byte)` method. Extended functionality available through optional interfaces:
   - `BeforeReceiveHandler`: Preprocess incoming data
   - `BeforeSendHandler`: Transform outgoing data
   - `AfterSendHandler`: Post-send notifications
   - `CloseHandler`: Connection close events
   - `PingGenerator`: Server-side ping generation
   - `ByeGenerator`: Disconnection notifications

3. **Network Adapters**: 
   - TCP: `tcp_conn.go` - Raw TCP socket support
   - WebSocket: `ws_conn.go` - WebSocket support with pluggable engines:
     - `ws_x.go`: golang.org/x/net/websocket adapter (default)
     - `ws_gorilla.go`: github.com/gorilla/websocket adapter
     - `wx_coder.go`: github.com/coder/websocket adapter  (former nhooyr.io/websocket)

4. **Queue System (`q.go`, `memory_q.go`)**: Interface-based offline message queue with built-in memory implementation. Supports TTL and deduplication.

5. **Device Management (`device.go`)**: Represents user devices with metadata and connection state.

### Key Design Patterns

- **Functional Options**: Hub configuration uses `WithHubConfig*` pattern for clean API
- **Interface Segregation**: Small, focused interfaces allow flexible implementation
- **Channel-Based Concurrency**: Hub uses channels for thread-safe message passing
- **Pluggable Components**: Queue and WebSocket engines are interface-based for easy replacement

### Important Considerations

- Single sign-on (SSO) ensures one connection per user ID
- Server ping or read timeout must be configured to prevent socket leaks
- All handlers are optional except the core Actor interface
- Tests use Ginkgo/Gomega BDD framework