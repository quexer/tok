/**
 * connection wrapper
 */

package tok

import (
	"code.google.com/p/go.net/websocket"
	"log"
	"net"
	"time"
)

type connection struct {
	uid     interface{}
	adapter ConAdapter
	ticker  *time.Ticker
	ch      chan []byte
	actor   Actor
}

type conState struct {
	con    *connection
	online bool
}

//ConAdapter if adapter for real connection.
//For now, net.Conn and  websocket.Conn are supported.
//This interface is useful for building test application
type ConAdapter interface {
	Read() ([]byte, error) //Read payload data from real connection. Unpack basic data frame
	Write([]byte) error    //Write payload data to real connection. Pack data with basic frame
	Close()                //Close the real connection
}

//BuildConAdapter build ConAdapter using real connection
//For now, net.Conn and  websocket.Conn are supported.
func BuildConAdapter(conn interface{}) ConAdapter {
	switch conn.(type) {
	case net.Conn:
		return &tcpAdapter{conn: conn.(net.Conn)}
	case *websocket.Conn:
		return &wsAdapter{conn: conn.(*websocket.Conn)}
	default:
		log.Fatal("not supported", conn)
		return nil
	}
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
			if b := conn.actor.Ping(); b != nil {
				conn.innerWrite(chState, b)
			}
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
		actor:   hub.actor,
	}

	hub.chConState <- &conState{conn, true}
	go conn.write(hub.chConState)
	conn.read(hub.chConState, hub.chUp)
}
