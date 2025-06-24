/**
 * websocket connection adapter
 */

package tok

import (
	"log"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
)

type wsAdapter struct {
	conn *websocket.Conn
	txt  bool
}

func (p *wsAdapter) Read() ([]byte, error) {
	if ReadTimeout > 0 {
		if err := p.conn.SetReadDeadline(time.Now().Add(ReadTimeout)); err != nil {
			log.Println("[warning] setting ws read deadline: ", err)
			return nil, err
		}
	}

	if p.txt {
		var s string
		err := websocket.Message.Receive(p.conn, &s)
		return []byte(s), err
	}

	var b []byte
	err := websocket.Message.Receive(p.conn, &b)
	return b, err

}

func (p *wsAdapter) Write(b []byte) error {
	if err := p.conn.SetWriteDeadline(time.Now().Add(WriteTimeout)); err != nil {
		log.Println("[warning] setting ws write deadline: ", err)
		return err
	}

	if p.txt {
		return websocket.Message.Send(p.conn, string(b))
	}

	return websocket.Message.Send(p.conn, b)
}

func (p *wsAdapter) Close() error {
	return p.conn.Close()
}

func (p *wsAdapter) ShareConn(adapter conAdapter) bool {
	wsAdp, ok := adapter.(*wsAdapter)
	if !ok {
		return false
	}
	return p.conn == wsAdp.conn
}

type WsHandler struct {
	hub       *Hub
	hubConfig *HubConfig // If config is not nil, a new hub will be created and replace old one
	txt       bool       // If txt is true web socket will serve text frame, otherwise serve binary frame
	auth      WsAuthFunc // auth function is used for user authorization
}

func (p *WsHandler) Handler() websocket.Handler {
	return websocket.Handler(func(ws *websocket.Conn) {
		adapter := &wsAdapter{conn: ws, txt: p.txt}

		if dv, err := p.auth(ws.Request()); err != nil {
			log.Printf("websocket auth err: %+v", err)
			adapter.Close()
		} else {
			p.hub.initConnection(dv, adapter)
		}
	})
}

// CreateWsHandler create web socket http handler with hub.
// auth function is used for user authorization
// Return http handler
func CreateWsHandler(auth WsAuthFunc, opts ...WsHandlerOption) (*Hub, http.Handler) {
	wsh := &WsHandler{
		hub:       nil,
		hubConfig: nil,
		txt:       true,
		auth:      auth,
	}

	for _, opt := range opts {
		opt(wsh)
	}

	if wsh.hubConfig != nil {
		wsh.hub = createHub(wsh.hubConfig.Actor, wsh.hubConfig.Q, wsh.hubConfig.Sso, wsh.hubConfig.ServerPingInterval)
	}

	if wsh.hub == nil {
		log.Fatal("hub is needed")
	}

	return wsh.hub, wsh.Handler()
}

// WsAuthFunc websocket auth function, return Device interface
// parameter is the initial websocket request
type WsAuthFunc func(*http.Request) (*Device, error)
