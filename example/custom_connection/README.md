# Custom Connection Examples

This directory contains examples of how to implement custom connection types for the tok library.

## Overview

The tok library now supports custom connection types through the `ConAdapter` interface. This allows you to integrate any transport protocol (QUIC, Unix sockets, named pipes, etc.) with tok's messaging system.

## ConAdapter Interface

To create a custom connection type, implement the `ConAdapter` interface:

```go
type ConAdapter interface {
    Read() ([]byte, error)
    Write(data []byte) error
    Close() error
    ShareConn(adapter ConAdapter) bool
}
```

### Method Requirements

- **Read()**: Should block until a complete message is available. Must handle message framing/delimiting.
- **Write()**: Must be thread-safe as it can be called concurrently. Should handle message framing.
- **Close()**: Should clean up resources and cause Read/Write to return errors.
- **ShareConn()**: Used for SSO mode to detect if two adapters share the same underlying connection.

## Examples

### 1. Unix Socket Adapter (`main.go`)

Shows how to implement a Unix domain socket adapter with simple length-prefixed framing:
- 4-byte header containing message length (big-endian)
- Followed by message payload

### 2. QUIC Adapter (`quic_example.go`)

Demonstrates QUIC integration:
- Uses QUIC streams for multiplexing
- Shows how to handle connection-level vs stream-level operations
- Includes authentication flow

### 3. Compressed Adapter (`quic_example.go`)

Shows how to wrap another adapter to add compression/encryption.

## Usage Pattern

1. **Implement ConAdapter** for your connection type
2. **Handle connection acceptance** in your own server loop
3. **Authenticate** the connection using your own logic
4. **Register** with hub using `hub.RegisterConnection(device, adapter)`

## Example Usage

```go
// Create your adapter
adapter := &MyCustomAdapter{conn: someConnection}

// Authenticate (your own logic)
device := authenticate(adapter)

// Register with hub
hub.RegisterConnection(device, adapter)
```

## Thread Safety

- The `Write` method must be thread-safe
- `Read` is called from a single goroutine
- Use appropriate locking in your adapter implementation

## Best Practices

1. **Message Framing**: Always implement proper message delimiting (length-prefix, delimiter, etc.)
2. **Error Handling**: Return appropriate errors from Read/Write when connection is closed
3. **Resource Cleanup**: Properly close underlying connections in Close()
4. **Timeouts**: Consider implementing read/write timeouts for robustness
5. **Authentication**: Perform authentication before registering with hub

## Testing

Test your adapter by:
1. Implementing a simple echo actor
2. Sending messages through your custom connection
3. Verifying message delivery and connection lifecycle