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

// ConAdapter is adapter for real connection.
// For now, net.Conn and websocket.Conn are supported.
// This interface is useful for building test application
// todo should export this interface, move to internal package
type ConAdapter interface {
	Read() ([]byte, error)             // Read payload data from real connection. Unpack from basic data frame
	Write([]byte) error                // Write payload data to real connection. Pack into basic data frame
	Close() error                      // Close the real connection
	ShareConn(adapter ConAdapter) bool // if two adapters share one net connection (tcp/ws)
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
