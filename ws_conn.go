/**
 * x websocket connection adapter
 */

package tok

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	coderws "github.com/coder/websocket"
	gorillaws "github.com/gorilla/websocket"
	"go.opentelemetry.io/otel/codes"
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
		_, span := p.hub.inst.tracer.Start(ws.Request().Context(), "tok.Auth")
		defer span.End()

		adapter := &xWsAdapter{
			conn:         ws,
			txt:          p.txt,
			writeTimeout: p.hub.config.writeTimeout,
			readTimeout:  p.hub.config.readTimeout,
		}

		if dv, err := p.auth(ws.Request()); err != nil {
			slog.Warn("websocket auth err", "err", err)
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			_ = adapter.Close()
		} else {
			p.hub.RegisterConnection(p.hub.ctx, dv, adapter)
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
		_, span := p.hub.inst.tracer.Start(r.Context(), "tok.Auth")
		defer span.End()

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Warn("gorilla websocket upgrade err", "err", err)
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			return
		}

		adapter := &gorillaWsAdapter{
			conn:         conn,
			txt:          p.txt,
			writeTimeout: p.hub.config.writeTimeout,
			readTimeout:  p.hub.config.readTimeout,
		}

		if dv, err := p.auth(r); err != nil {
			slog.Warn("gorilla websocket auth err", "err", err)
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			_ = adapter.Close()
		} else {
			p.hub.RegisterConnection(p.hub.ctx, dv, adapter)
		}
	}
}

// hdlFromCoderWebSocket returns a coder/websocket handler function that handles incoming websocket connections.
func (p *WsHandler) hdlFromCoderWebSocket() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, span := p.hub.inst.tracer.Start(r.Context(), "tok.Auth")
		defer span.End()

		conn, err := coderws.Accept(w, r, nil)
		if err != nil {
			slog.Warn("coder websocket accept err", "err", err)
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			return
		}

		adapter := &coderWsAdapter{
			conn:         conn,
			ctx:          p.hub.ctx,
			txt:          p.txt,
			writeTimeout: p.hub.config.writeTimeout,
			readTimeout:  p.hub.config.readTimeout,
		}

		if dv, err := p.auth(r); err != nil {
			slog.Warn("coder websocket auth err", "err", err)
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			_ = adapter.Close()
		} else {
			p.hub.RegisterConnection(p.hub.ctx, dv, adapter)
		}
	}
}

func (p *WsHandler) hdl() http.Handler {
	switch p.engine {
	case WsEngineGorilla:
		return p.hdlFromGorillaWebSocket()
	case WsEngineCoder:
		return p.hdlFromCoderWebSocket()
	default:
		return p.hdlFromXwebSocket()
	}
}

// CreateWsHandler create websocket http handler
// auth function is used for user authorization
// auth is required and must not be nil.
// Return hub and http handler
func CreateWsHandler(ctx context.Context, auth WsAuthFunc, opts ...WsHandlerOption) (*Hub, http.Handler, error) {
	if auth == nil {
		return nil, nil, ErrAuthRequired
	}

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
		var err error
		wsh.hub, err = createHub(ctx, wsh.hubConfig)
		if err != nil {
			return nil, nil, fmt.Errorf("create hub: %w", err)
		}
		wsh.hubConfig = nil // config consumed; avoid stale reference
	}

	if wsh.hub == nil {
		return nil, nil, errors.New("hub is needed")
	}

	return wsh.hub, wsh.hdl(), nil
}

// WsAuthFunc websocket auth function, return Device interface
// parameter is the initial websocket request
type WsAuthFunc func(*http.Request) (*Device, error)
