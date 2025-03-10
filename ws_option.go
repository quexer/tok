package tok

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
