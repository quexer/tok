/**
 * connection wrapper
 */

package tok

import (
	//	"log"
	"time"
)

type func_ping func() []byte

type connection struct {
	uid     interface{}
	adapter ConAdapter
	ticker  *time.Ticker
	ch      chan []byte
	ping    func_ping
}

type conState struct {
	con    *connection
	online bool
}

type ConAdapter interface {
	Read() ([]byte, error)
	Write([]byte) error
	Close()
}

func (conn *connection) read(chState chan<- *conState, chUp chan<- *frame) {
	for {
		b, err := conn.adapter.Read()
		if err != nil {
			//			log.Println("read err", err)
			chState <- &conState{con: conn, online: false}
			break
		}
		chUp <- &frame{uid: conn.uid, data: b}
	}
}

func (conn *connection) write(chState chan<- *conState) {
	for {
		select {
		case b := <-conn.ch:
			if b == nil {
				conn.ticker.Stop()
				conn.adapter.Close()
				return
			}
			//			log.Println("down msg for ", conn)
			conn.innerWrite(chState, b)
		case <-conn.ticker.C:
			conn.innerWrite(chState, conn.ping())
		}
	}

}

func (conn *connection) innerWrite(chState chan<- *conState, b []byte) {
	if err := conn.adapter.Write(b); err != nil {
		chState <- &conState{con: conn, online: false}
	}
}

func (conn *connection) close() {
	//	log.Println("close down channel", conn)
	close(conn.ch)
}

//block on read
func initConnection(uid interface{}, adapter ConAdapter, hub *Hub) {
	//	log.Println("new conection ", uid)

	conn := &connection{
		uid:     uid,
		adapter: adapter,
		ch:      make(chan []byte, 256),
		ticker:  time.NewTicker(30 * 1e9),
		ping:    hub.actor.Ping,
	}

	hub.chConState <- &conState{conn, true}
	go conn.write(hub.chConState)
	conn.read(hub.chConState, hub.chUp)
}
