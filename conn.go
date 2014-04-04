/**
 * connection wrapper
 */

package tok

import "time"

type connection struct {
	uid     interface{}
	adapter conAdapter
	ticker  *time.Ticker
	ch      chan []byte
	actor   Actor
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

func (conn *connection) read(hub *Hub, chUp chan<- *frame) {
	for {
		b, err := conn.adapter.Read()
		if err != nil {
			//			log.Println("read err", err)
			hub.stateChange(conn, false)
			break
		}
		chUp <- &frame{uid: conn.uid, data: b}
	}
}

func (conn *connection) write(hub *Hub) {
	for {
		select {
		case b := <-conn.ch:
			if b == nil {
				conn.ticker.Stop()
				conn.adapter.Close()
				return
			}
			//			log.Println("down msg for ", conn)
			conn.innerWrite(hub, b)
		case <-conn.ticker.C:
			if b := conn.actor.Ping(); b != nil {
				conn.innerWrite(hub, b)
			}
		}
	}

}

func (conn *connection) innerWrite(hub *Hub, b []byte) {
	if err := conn.adapter.Write(b); err != nil {
		hub.stateChange(conn, false)
	}
}

func (conn *connection) close() {
	//	log.Println("close down channel", conn)
	close(conn.ch)
}

//block on read
func initConnection(uid interface{}, adapter conAdapter, hub *Hub) {
	//	log.Println("new conection ", uid)

	conn := &connection{
		uid:     uid,
		adapter: adapter,
		ch:      make(chan []byte, 256),
		ticker:  time.NewTicker(30 * 1e9),
		actor:   hub.actor,
	}

	hub.stateChange(conn, true)
	go conn.write(hub)
	conn.read(hub, hub.chUp)
}
