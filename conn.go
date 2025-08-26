/**
 * connection wrapper
 */

package tok

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
)

//go:generate mockgen -destination=mocks/conn.go -package=mocks . ConAdapter

// connection represents an abstract connection with thread-safe operations.
//
// Thread-safety:
// - wLock: Ensures write operations are serialized (Write method only)
// - closed: Uses atomic operations for lock-free status check
// - offlineTriggered: Uses atomic operations for exactly-once semantics
type connection struct {
	// wLock ensures write operations are serialized
	wLock            sync.Mutex
	dv               *Device            // device of this connection
	adapter          ConAdapter         // real connection adapter
	hub              *Hub               // hub of this connection
	cancelFunc       context.CancelFunc // cancel function for ping goroutine
	closed           int32              // connection closed flag (atomic: 0=open, 1=closed)
	offlineTriggered int32              // ensure offline state change is triggered only once (atomic)
}

// conState is the state of connection
type conState struct {
	con    *connection
	online bool
}

// ShareConn check if two connections share the same underline connection
func (conn *connection) ShareConn(other *connection) bool {
	return conn.adapter.ShareConn(other.adapter)
}

// ConAdapter is the adapter interface for real connections.
// Users can implement this interface to support custom connection types beyond the built-in TCP and WebSocket.
//
// Implementations must be thread-safe for Write operations, as Write may be called concurrently.
// Read operations are called sequentially from a single goroutine.
type ConAdapter interface {
	// Read reads the next message from the connection.
	// Read should block until a message is available or an error occurs.
	// The returned data should be the complete message payload (not including any protocol framing).
	Read() ([]byte, error)

	// Write writes a message to the connection.
	// Write must be thread-safe as it may be called concurrently.
	// The data parameter is the complete message payload to send.
	Write(data []byte) error

	// Close closes the connection.
	// After Close is called, all Read and Write operations should return errors.
	Close() error

	// ShareConn returns true if this adapter shares the same underlying connection with another adapter.
	// This is used for connection deduplication in SSO (Single Sign-On) mode.
	ShareConn(adapter ConAdapter) bool
}

func (conn *connection) uid() interface{} {
	return conn.dv.UID()
}

// triggerOffline triggers offline state change only once
func (conn *connection) triggerOffline() {
	if atomic.CompareAndSwapInt32(&conn.offlineTriggered, 0, 1) {
		conn.hub.stateChange(conn, false)
	}
}

func (conn *connection) isClosed() bool {
	return atomic.LoadInt32(&conn.closed) == 1
}

func (conn *connection) readLoop() {
	for {
		if conn.isClosed() {
			return
		}

		b, err := conn.adapter.Read()
		if err != nil {
			slog.Debug("read err", "err", err)
			conn.triggerOffline()
			return
		}
		conn.hub.receive(conn.dv, b)
	}
}

func (conn *connection) close() {
	// Use atomic compare-and-swap to ensure close is called only once
	if !atomic.CompareAndSwapInt32(&conn.closed, 0, 1) {
		return
	}

	// Now we have exclusive access to close the connection
	if conn.cancelFunc != nil {
		conn.cancelFunc() // cancel ping goroutine
	}
	_ = conn.adapter.Close()
}

func (conn *connection) Write(b []byte) error {
	conn.wLock.Lock()
	defer conn.wLock.Unlock()

	if conn.isClosed() {
		return errors.New("can't write to closed connection")
	}

	if err := conn.adapter.Write(b); err != nil {
		conn.triggerOffline()
		return err
	}
	return nil
}
