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

	// Define the BeforeReceive function
	beforeReceive := func(dv *tok.Device, data []byte) ([]byte, error) {
		slog.Info("BeforeReceive", "dv", &dv, "data", data)
		return data, nil
	}

	// Define the BeforeSend function
	beforeSend := func(dv *tok.Device, data []byte) ([]byte, error) {
		slog.Info("BeforeSend", "dv", &dv, "data", data)
		return data, nil
	}

	// Define the AfterSend function (use functional option for AfterSend functionality)
	afterSend := func(dv *tok.Device, data []byte) {
		slog.Info("AfterSend via functional option", "dv", &dv, "data", data)
	}

	hc := tok.NewHubConfig(&simpleActor{},
		tok.WithHubConfigServerPingInterval(2*time.Second),
		tok.WithHubConfigPingProducer(&SimplePingProducer{}),
		tok.WithHubConfigBeforeReceive(beforeReceive),
		tok.WithHubConfigBeforeSend(beforeSend),
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

func (p *simpleActor) Bye(kicker *tok.Device, reason string, dv *tok.Device) []byte {
	return nil
}

// SimplePingProducer implements PingGenerator interface
type SimplePingProducer struct{}

func (p *SimplePingProducer) Ping() []byte {
	slog.Info("Send Ping")
	return []byte("ping")
}
