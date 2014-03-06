/**
 * "talk"
 */

package tok

import (
	"code.google.com/p/go.net/websocket"
	"log"
	"net"
	"net/http"
)

type Auth func(*http.Request) (interface{}, error)

type Actor interface {
	OnReceive(uid interface{}, data []byte) ([]interface{}, []byte, error)
	Ping() []byte //build ping data.  ping will be ignored if return nil
	Bye(reason string) []byte
}

type Queue interface {
	Enq(uid interface{}, data []byte) error
	Deq(uid interface{}) ([]byte, error)
	Len(uid interface{}) (int, error)
}

func CreateHub(actor Actor, q Queue, sso bool) *Hub {
	hub := &Hub{
		sso:          sso,
		actor:        actor,
		q:            q,
		cons:         make(map[interface{}][]*connection),
		chUp:         make(chan *frame),
		chDown:       make(chan *frame),
		chDown2:      make(chan *frame),
		chConState:   make(chan *conState),
		chReadSignal: make(chan interface{}, 100),
	}
	go hub.run()
	return hub
}

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
