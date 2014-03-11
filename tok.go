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

type Actor interface {
	Auth(r *http.Request) (interface{}, error) //auth against http request. return uid if auth success
	OnReceive(uid interface{}, data []byte)
	Ping() []byte //build ping data.  ping will be ignored if return nil
	Bye(reason string) []byte
}

type Queue interface {
	Enq(uid interface{}, data []byte) error
	Deq(uid interface{}) ([]byte, error)
	Len(uid interface{}) (int, error)
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
