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
}

func (p *wsAdapter) Read() ([]byte, error) {
	var b []byte
	err := websocket.Message.Receive(p.conn, &b)
	return b, err
}

func (p *wsAdapter) Write(b []byte) error {
	return websocket.Message.Send(p.conn, b)
}

func (p *wsAdapter) Close() {
	p.conn.Close()
}

func CreateWsHandler(auth Auth, hub *Hub) http.Handler {
	return websocket.Handler(func(ws *websocket.Conn) {
		adapter := &wsAdapter{conn: ws}
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
