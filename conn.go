/**
 * connection wrapper
 */

package tok

import (
	"errors"
	"log"
	"sync"
	"time"
)

var (
	// ReadTimeout read timeout duration
	ReadTimeout time.Duration
	// WriteTimeout write timeout duration
	WriteTimeout = time.Minute
	// AuthTimeout auth timeout duration
	AuthTimeout = time.Second * 5
	// ServerPingInterval server ping interval duration
	ServerPingInterval = time.Second * 30
)

// abstract connection,
type connection struct {
	sync.RWMutex
	wLock   sync.Mutex
	dv      *Device
	adapter conAdapter
	hub     *Hub
	closed  bool
}

type conState struct {
	con    *connection
	online bool
}

func (conn *connection) ShareConn(other *connection) bool {
	return conn.adapter.ShareConn(other.adapter)
}

// conAdapter is adapter for real connection.
// For now, net.Conn and websocket.Conn are supported.
// This interface is useful for building test application
type conAdapter interface {
	Read() ([]byte, error)             // Read payload data from real connection. Unpack from basic data frame
	Write([]byte) error                // Write payload data to real connection. Pack into basic data frame
	Close()                            // Close the real connection
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
	conn.adapter.Close()
}

func (conn *connection) Write(b []byte) error {
	conn.wLock.Lock()
	defer conn.wLock.Unlock()

	if conn.isClosed() {
		return errors.New("Can't write to closed connection")
	}

	if err := conn.adapter.Write(b); err != nil {
		conn.hub.stateChange(conn, false)
		return err
	}
	return nil
}

func initConnection(dv *Device, adapter conAdapter, hub *Hub) {
	conn := &connection{
		dv:      dv,
		adapter: adapter,
		hub:     hub,
	}

	hub.stateChange(conn, true)

	// start server ping loop if necessary
	if hub.actor.Ping() != nil {
		ticker := time.NewTicker(ServerPingInterval)
		go func() {
			for range ticker.C {
				if conn.isClosed() {
					ticker.Stop()
					return
				}
				b, err := hub.actor.BeforeSend(dv, hub.actor.Ping())
				if err == nil {
					if b == nil {
						b = hub.actor.Ping()
					}
					if err := conn.Write(b); err != nil {
						log.Println("[tok] write ping error", err)
					}
				}
			}
		}()
	}

	// block on read
	conn.readLoop()
}
