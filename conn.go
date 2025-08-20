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

// abstract connection,
type connection struct {
	sync.RWMutex
	wLock            sync.Mutex         // write lock
	dv               *Device            // device of this connection
	adapter          ConAdapter         // real connection adapter
	hub              *Hub               // hub of this connection
	closed           bool               // connection closed flag
	cancelFunc       context.CancelFunc // cancel function for ping goroutine
	offlineTriggered int32              // ensure offline state change is triggered only once
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
	conn.RLock()
	defer conn.RUnlock()
	return conn.closed
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
	conn.Lock()
	defer conn.Unlock()

	if conn.closed {
		return
	}

	conn.closed = true
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
