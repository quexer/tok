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
func initConnection(uid interface{}, adapter conAdapter, hub *Hub) {
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
