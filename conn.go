/**
 * connection wrapper
 */

package tok

import (
	"fmt"
	"sync"
	"time"
)

var (
	//READ_TIMEOUT read timeout duration
	READ_TIMEOUT         time.Duration
	//WRITE_TIMEOUT write timeout duration
	WRITE_TIMEOUT                      = time.Minute
	//AUTH_TIMEOUT auth timeout duration
	AUTH_TIMEOUT                       = time.Second * 5
	//SERVER_PING_INTERVAL server ping interval duration
	SERVER_PING_INTERVAL               = time.Second * 30
)

type connection struct {
	sync.RWMutex
	wLock   sync.Mutex
	uid     interface{}
	adapter conAdapter
	hub     *Hub
	closed  bool
}

type conState struct {
	con    *connection
	online bool
}

//conAdapter is adapter for real connection.
//For now, net.Conn and websocket.Conn are supported.
//This interface is useful for building test application
type conAdapter interface {
	Read() ([]byte, error) //Read payload data from real connection. Unpack from basic data frame
	Write([]byte) error    //Write payload data to real connection. Pack into basic data frame
	Close()                //Close the real connection
}

func (conn *connection) isClosed() bool {
	conn.RLock()
	defer conn.RUnlock()
	return conn.closed
}

func (conn *connection) readLoop(hub *Hub) {
	for {
		if conn.isClosed() {
			return
		}

		b, err := conn.adapter.Read()
		if err != nil {
			//			log.Println("read err", err)
			hub.stateChange(conn, false)
			return
		}
		hub.receive(conn.uid, b)
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
		return fmt.Errorf("Can't write to closed connection")
	}

	err := conn.adapter.Write(b)
	if err != nil {
		conn.hub.stateChange(conn, false)
	}
	return err
}

func initConnection(uid interface{}, adapter conAdapter, hub *Hub) {
	//	log.Println("new conection ", uid)

	conn := &connection{
		uid:     uid,
		adapter: adapter,
		hub:     hub,
	}

	hub.stateChange(conn, true)

	//start server ping loop if necessary
	if hub.actor.Ping() != nil {
		ticker := time.NewTicker(SERVER_PING_INTERVAL)
		go func() {
			for range ticker.C {
				if conn.isClosed() {
					ticker.Stop()
					return
				}
				b, err := hub.actor.BeforeSend(uid, hub.actor.Ping())
				if err == nil {
					if b == nil {
						b = hub.actor.Ping()
					}
					conn.Write(b)
				}
			}
		}()
	}

	//block on read
	conn.readLoop(hub)
}
