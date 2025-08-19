package tok

// WsEngine represents different WebSocket engine implementations
type WsEngine int

const (
	// WsEngineX uses golang.org/x/net/websocket (default)
	WsEngineX WsEngine = iota
	// WsEngineGorilla uses github.com/gorilla/websocket
	WsEngineGorilla
	// WsEngineCoder uses github.com/coder/websocket (former nhooyr.io/websocket)
	WsEngineCoder
	// Future engines can be easily added here, e.g.:
	// WsEngineCustom for custom implementations
)

type WsHandlerOption func(*WsHandler)

// WithWsHandlerTxt set txt mode for ws handler
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
