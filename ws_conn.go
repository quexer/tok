/**
 * websocket connection adapter
 */

package tok

import (
	"code.google.com/p/go.net/websocket"
	"log"
	"net/http"
)

type wsAdapter struct {
	conn *websocket.Conn
	txt  bool
}

func (p *wsAdapter) Read() ([]byte, error) {
	if p.txt {
		var s string
		err := websocket.Message.Receive(p.conn, &s)
		return []byte(s), err
	} else {
		var b []byte
		err := websocket.Message.Receive(p.conn, &b)
		return b, err
	}
}

func (p *wsAdapter) Write(b []byte) error {
	if p.txt {
		return websocket.Message.Send(p.conn, string(b))
	} else {
		return websocket.Message.Send(p.conn, b)
	}
}

func (p *wsAdapter) Close() {
	p.conn.Close()
}

//create binary ws handler
func CreateWsHandler(auth Auth, hub *Hub) http.Handler {
	return CreateWsHandlerEx(auth, hub, false)
}

//create ws handler, txt parameter is binary/text frame flag
func CreateWsHandlerEx(auth Auth, hub *Hub, txt bool) http.Handler {
	return websocket.Handler(func(ws *websocket.Conn) {
		adapter := &wsAdapter{conn: ws, txt: txt}
		r := ws.Request()
		uid, err := auth(r)
		if err != nil {
			log.Println("401", err)
			adapter.Write(hub.actor.Bye("unauthorized"))
			adapter.Close()
			return
		}
		//		log.Println("new ws connection for", uid)
		initConnection(uid, adapter, hub)
	})
}
