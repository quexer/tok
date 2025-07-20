package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/quexer/tok"
)

var (
	hub *tok.Hub
)

func main() {
	var hdl http.Handler

	// Define the BeforeReceive handler
	beforeReceiveHandler := &SimpleBeforeReceiveHandler{}

	// Define the BeforeSend handler
	beforeSendHandler := &SimpleBeforeSendHandler{}

	// Define the AfterSend function (use functional option for AfterSend functionality)
	afterSend := func(dv *tok.Device, data []byte) {
		slog.Info("AfterSend via functional option", "dv", &dv, "data", data)
	}

	hc := tok.NewHubConfig(&simpleActor{},
		tok.WithHubConfigServerPingInterval(2*time.Second),
		tok.WithHubConfigPingProducer(&SimplePingProducer{}),
		tok.WithHubConfigBeforeReceive(beforeReceiveHandler),
		tok.WithHubConfigBeforeSend(beforeSendHandler),
		// Use AfterSend via functional option (AfterSend method is no longer in Actor interface)
		tok.WithHubConfigAfterSend(afterSend),
	)

	authFunc := func(r *http.Request) (*tok.Device, error) {
		return tok.CreateDevice(fmt.Sprintf("%p", r), ""), nil
	}

	hub, hdl = tok.CreateWsHandler(authFunc, tok.WithWsHandlerHubConfig(hc))

	http.Handle("/ws", hdl)

	err := http.ListenAndServe(":8090", nil)
	if err != nil {
		log.Fatalf("Error starting HTTP server: %v", err)
	}

}

type simpleActor struct {
}

func (p *simpleActor) OnReceive(dv *tok.Device, data []byte) {
	slog.Info("OnReceive", "dv", &dv, "data", data)
	return
}

// SimpleBeforeReceiveHandler implements BeforeReceiveHandler interface
type SimpleBeforeReceiveHandler struct{}

func (h *SimpleBeforeReceiveHandler) BeforeReceive(dv *tok.Device, data []byte) ([]byte, error) {
	slog.Info("BeforeReceive", "dv", &dv, "data", data)
	return data, nil
}

// SimpleBeforeSendHandler implements BeforeSendHandler interface
type SimpleBeforeSendHandler struct{}

func (h *SimpleBeforeSendHandler) BeforeSend(dv *tok.Device, data []byte) ([]byte, error) {
	slog.Info("BeforeSend", "dv", &dv, "data", data)
	return data, nil
}

// SimplePingProducer implements PingGenerator interface
type SimplePingProducer struct{}

func (p *SimplePingProducer) Ping() []byte {
	slog.Info("Send Ping")
	return []byte("ping")
}
