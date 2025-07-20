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

	gorillaws "github.com/gorilla/websocket"
	xwebsocket "golang.org/x/net/websocket"
)

// xWsAdapter is an adapter for golang.org/x/net/websocket connections.
// It implements the conAdapter interface and provides unified read/write/timeout management for websockets.
type xWsAdapter struct {
	conn         *xwebsocket.Conn // Underlying x websocket connection
	txt          bool             // If true, use text frames; otherwise, use binary frames
	writeTimeout time.Duration    // Timeout for write operations
	readTimeout  time.Duration    // Timeout for read operations
}

func (p *xWsAdapter) Read() ([]byte, error) {
	if p.readTimeout > 0 {
		if err := p.conn.SetReadDeadline(time.Now().Add(p.readTimeout)); err != nil {
			return nil, fmt.Errorf("setting x ws read deadline err: %w", err)
		}
	}

	if p.txt {
		var s string
		err := xwebsocket.Message.Receive(p.conn, &s)
		return []byte(s), err
	}

	var b []byte
	err := xwebsocket.Message.Receive(p.conn, &b)
	return b, err

}

func (p *xWsAdapter) Write(b []byte) error {
	if err := p.conn.SetWriteDeadline(time.Now().Add(p.writeTimeout)); err != nil {
		return fmt.Errorf("setting x ws write deadline failed: %w", err)
	}

	if p.txt {
		return xwebsocket.Message.Send(p.conn, string(b))
	}

	return xwebsocket.Message.Send(p.conn, b)
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
	hub        *Hub
	hubConfig  *HubConfig // If config is not nil, a new hub will be created and replace old one
	txt        bool       // If txt is true web socket will serve text frame, otherwise serve binary frame
	auth       WsAuthFunc // auth function is used for user authorization
	useGorilla bool       // If true, use Gorilla WebSocket; otherwise, use x/net/websocket
}

// hdlFromXwebSocket returns an x/web/websocket handler function that handles incoming websocket connections.
func (p *WsHandler) hdlFromXwebSocket() xwebsocket.Handler {
	return func(ws *xwebsocket.Conn) {
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

// hdlFromGorillaWebSocket returns a gorilla/websocket handler function that handles incoming websocket connections.
func (p *WsHandler) hdlFromGorillaWebSocket() http.HandlerFunc {
	upgrader := gorillaws.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow connections from any origin for now
		},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Warn("gorilla websocket upgrade err", "err", err)
			return
		}

		adapter := &gorillaWsAdapter{
			conn:         conn,
			txt:          p.txt,
			writeTimeout: p.hubConfig.writeTimeout,
			readTimeout:  p.hubConfig.readTimeout,
		}

		if dv, err := p.auth(r); err != nil {
			slog.Warn("gorilla websocket auth err", "err", err)
			_ = adapter.Close()
		} else {
			p.hub.initConnection(dv, adapter)
		}
	}
}

// CreateWsHandler create websocket http handler
// auth function is used for user authorization
// Return hub and http handler
func CreateWsHandler(auth WsAuthFunc, opts ...WsHandlerOption) (*Hub, http.Handler) {
	wsh := &WsHandler{
		hub:        nil,
		hubConfig:  nil,
		txt:        true,
		auth:       auth,
		useGorilla: false, // Default to x/net/websocket for backward compatibility
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

	if wsh.useGorilla {
		return wsh.hub, wsh.hdlFromGorillaWebSocket()
	}
	return wsh.hub, wsh.hdlFromXwebSocket()
}

// WsAuthFunc websocket auth function, return Device interface
// parameter is the initial websocket request
type WsAuthFunc func(*http.Request) (*Device, error)
