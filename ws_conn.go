/**
 * x websocket connection adapter
 */

package tok

import (
	"context"
	"log"
	"log/slog"
	"net/http"

	coderws "github.com/coder/websocket"
	gorillaws "github.com/gorilla/websocket"
	xwebsocket "golang.org/x/net/websocket"
)

type WsHandler struct {
	hub       *Hub
	hubConfig *HubConfig // If config is not nil, a new hub will be created and replace old one
	txt       bool       // If txt is true web socket will serve text frame, otherwise serve binary frame
	auth      WsAuthFunc // auth function is used for user authorization
	engine    WsEngine   // WebSocket engine to use
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

// hdlFromCoderWebSocket returns a coder/websocket handler function that handles incoming websocket connections.
func (p *WsHandler) hdlFromCoderWebSocket() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Accept WebSocket connection with default options
		conn, err := coderws.Accept(w, r, nil)
		if err != nil {
			slog.Warn("coder websocket accept err", "err", err)
			return
		}

		adapter := &coderWsAdapter{
			conn:         conn,
			ctx:          context.Background(),
			txt:          p.txt,
			writeTimeout: p.hubConfig.writeTimeout,
			readTimeout:  p.hubConfig.readTimeout,
		}

		if dv, err := p.auth(r); err != nil {
			slog.Warn("coder websocket auth err", "err", err)
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
		hub:       nil,
		hubConfig: nil,
		txt:       true,
		auth:      auth,
		engine:    WsEngineX, // Default to x/net/websocket for backward compatibility
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

	switch wsh.engine {
	case WsEngineGorilla:
		return wsh.hub, wsh.hdlFromGorillaWebSocket()
	case WsEngineCoder:
		return wsh.hub, wsh.hdlFromCoderWebSocket()
	default:
		return wsh.hub, wsh.hdlFromXwebSocket()
	}
}

// WsAuthFunc websocket auth function, return Device interface
// parameter is the initial websocket request
type WsAuthFunc func(*http.Request) (*Device, error)
