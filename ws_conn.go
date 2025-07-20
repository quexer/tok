/**
 * x websocket connection adapter
 */

package tok

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
)

type xWsAdapter struct {
	conn         *websocket.Conn
	txt          bool
	writeTimeout time.Duration
	readTimeout  time.Duration
}

func (p *xWsAdapter) Read() ([]byte, error) {
	if p.readTimeout > 0 {
		if err := p.conn.SetReadDeadline(time.Now().Add(p.readTimeout)); err != nil {
			return nil, fmt.Errorf("setting ws read deadline err: %w", err)
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

func (p *xWsAdapter) Write(b []byte) error {
	if err := p.conn.SetWriteDeadline(time.Now().Add(p.writeTimeout)); err != nil {
		return fmt.Errorf("setting ws write deadline failed: %w", err)
	}

	if p.txt {
		return websocket.Message.Send(p.conn, string(b))
	}

	return websocket.Message.Send(p.conn, b)
}

func (p *xWsAdapter) Close() error {
	return p.conn.Close()
}

func (p *xWsAdapter) ShareConn(adapter conAdapter) bool {
	wsAdapter, ok := adapter.(*xWsAdapter)
	if !ok {
		return false
	}
	return p.conn == wsAdapter.conn
}

type WsHandler struct {
	hub       *Hub
	hubConfig *HubConfig // If config is not nil, a new hub will be created and replace old one
	txt       bool       // If txt is true web socket will serve text frame, otherwise serve binary frame
	auth      WsAuthFunc // auth function is used for user authorization
}

func (p *WsHandler) Handler() websocket.Handler {
	return func(ws *websocket.Conn) {
		adapter := &xWsAdapter{
			conn:         ws,
			txt:          p.txt,
			writeTimeout: p.hubConfig.writeTimeout,
		}

		if dv, err := p.auth(ws.Request()); err != nil {
			slog.Warn("websocket auth err", "err", err)
			_ = adapter.Close()
		} else {
			p.hub.initConnection(dv, adapter)
		}
	}
}

// CreateWsHandler create x websocket http handler
// auth function is used for user authorization
// Return hub and http handler
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
		wsh.hub = createHub(wsh.hubConfig)
	}

	if wsh.hub == nil {
		log.Fatal("hub is needed")
	}

	return wsh.hub, wsh.Handler()
}

// WsAuthFunc websocket auth function, return Device interface
// parameter is the initial websocket request
type WsAuthFunc func(*http.Request) (*Device, error)
