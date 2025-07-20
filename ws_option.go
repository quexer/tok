package tok

// WsEngine represents different WebSocket engine implementations
type WsEngine int

const (
	// WsEngineXNet uses golang.org/x/net/websocket (default)
	WsEngineXNet WsEngine = iota
	// WsEngineGorilla uses github.com/gorilla/websocket
	WsEngineGorilla
	// Future engines can be easily added here, e.g.:
	// WsEngineNhooyr for nhooyr.io/websocket
	// WsEngineCustom for custom implementations
)

type WsHandlerOption func(*WsHandler)

// WithWsHandlerTxt set txt for ws handler
func WithWsHandlerTxt(txt bool) WsHandlerOption {
	return func(h *WsHandler) {
		h.txt = txt
	}
}

// WithWsHandlerHub set hub for ws handler, if hubConfig is nil, hub will be used
func WithWsHandlerHub(hub *Hub) WsHandlerOption {
	return func(h *WsHandler) {
		if h.hubConfig == nil {
			h.hub = hub
		}
	}
}

// WithWsHandlerHubConfig set hub config for ws handler
func WithWsHandlerHubConfig(hc *HubConfig) WsHandlerOption {
	return func(h *WsHandler) {
		h.hubConfig = hc
	}
}

// WithWsHandlerEngine sets the websocket engine for ws handler
func WithWsHandlerEngine(engine WsEngine) WsHandlerOption {
	return func(h *WsHandler) {
		h.engine = engine
	}
}

// WithWsHandlerGorilla set whether to use Gorilla WebSocket instead of x/net/websocket
// Deprecated: Use WithWsHandlerEngine(WsEngineGorilla) or WithWsHandlerEngine(WsEngineXNet) instead
func WithWsHandlerGorilla(useGorilla bool) WsHandlerOption {
	return func(h *WsHandler) {
		if useGorilla {
			h.engine = WsEngineGorilla
		} else {
			h.engine = WsEngineXNet
		}
	}
}
