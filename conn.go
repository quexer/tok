/**
 * connection wrapper
 */

package tok

import (
	"errors"
	"sync"
	"time"
)

var (
	// ReadTimeout read timeout duration
	ReadTimeout time.Duration
)

// abstract connection,
type connection struct {
	sync.RWMutex
	wLock   sync.Mutex // write lock
	dv      *Device    // device of this connection
	adapter conAdapter // real connection adapter
	hub     *Hub       // hub of this connection
	closed  bool       // connection closed flag
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

// conAdapter is adapter for real connection.
// For now, net.Conn and websocket.Conn are supported.
// This interface is useful for building test application
type conAdapter interface {
	Read() ([]byte, error)             // Read payload data from real connection. Unpack from basic data frame
	Write([]byte) error                // Write payload data to real connection. Pack into basic data frame
	Close() error                      // Close the real connection
	ShareConn(adapter conAdapter) bool // if two adapters share one net connection (tcp/ws)
}

func (conn *connection) uid() interface{} {
	return conn.dv.UID()
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
			//			log.Println("read err", err)
			conn.hub.stateChange(conn, false)
			return
		}
		conn.hub.receive(conn.dv, b)
	}
}

func (conn *connection) close() {
	conn.Lock()
	defer conn.Unlock()

	conn.closed = true
	_ = conn.adapter.Close()
}

func (conn *connection) Write(b []byte) error {
	conn.wLock.Lock()
	defer conn.wLock.Unlock()

	if conn.isClosed() {
		return errors.New("can't write to closed connection")
	}

	if err := conn.adapter.Write(b); err != nil {
		conn.hub.stateChange(conn, false)
		return err
	}
	return nil
}
