/**
 * websocket connection adapter
 */

package tok

import (
	"code.google.com/p/go.net/websocket"
	"log"
	"net/http"
	"time"
)

type wsAdapter struct {
	conn *websocket.Conn
	txt  bool
}

func (p *wsAdapter) Read() ([]byte, error) {
	if READ_TIMEOUT > 0 {
		if err := p.conn.SetReadDeadline(time.Now().Add(READ_TIMEOUT)); err != nil {
			log.Println("[warning] setting ws read deadline: ", err)
			return nil, err
		}
	}

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
	if err := p.conn.SetWriteDeadline(time.Now().Add(WRITE_TIMEOUT)); err != nil {
		log.Println("[warning] setting ws write deadline: ", err)
		return err
	}

	if p.txt {
		return websocket.Message.Send(p.conn, string(b))
	} else {
		return websocket.Message.Send(p.conn, b)
	}
}

func (p *wsAdapter) Close() {
	p.conn.Close()
}

//CreateWsHandler create web socket http handler with hub.
//If config is not nil, a new hub will be created and replace old one
//If txt is true web socket will serve text frame, otherwise serve binary frame
//Return http handler
func CreateWsHandler(hub *Hub, config *HubConfig, txt bool) (*Hub, http.Handler) {
	if config != nil {
		hub = createHub(config.Actor, config.Q, config.Sso)
	}

	if hub == nil {
		log.Fatal("hub is needed")
	}

	return hub, websocket.Handler(func(ws *websocket.Conn) {
		adapter := &wsAdapter{conn: ws, txt: txt}
		r := ws.Request()
		uid, err := hub.actor.Auth(r)
		if err != nil {
			log.Println("401", err)
			adapter.Write(hub.actor.Bye(uid, "unauthorized"))
			adapter.Close()
			return
		}
		//		log.Println("new ws connection for", uid)
		initConnection(uid, adapter, hub)
	})
}
