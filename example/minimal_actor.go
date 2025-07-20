package main

import (
	"fmt"
	"log/slog"

	"github.com/quexer/tok"
)

// MinimalActor demonstrates an actor with only required methods
// AfterSend functionality is available via functional options only
type MinimalActor struct{}

func (a *MinimalActor) OnReceive(dv *tok.Device, data []byte) {
	slog.Info("MinimalActor.OnReceive", "dv", &dv, "data", string(data))
}

// SimplePingProducer implements the PingGenerator interface
type SimplePingProducer struct{}

func (p *SimplePingProducer) Ping() []byte {
	return []byte("ping")
}

// SimpleByeGenerator implements the ByeGenerator interface
type SimpleByeGenerator struct{}

func (b *SimpleByeGenerator) Bye(kicker *tok.Device, reason string, dv *tok.Device) []byte {
	return []byte(fmt.Sprintf("bye: %s", reason))
}

func main() {
	// Example of creating a hub with AfterSend via functional option
	ExampleWithAfterSendOption()

	// Example of creating a hub with CloseHandler via functional option
	ExampleWithCloseHandlerOption()
	
	// Example of creating a hub with ByeGenerator via functional option
	ExampleWithByeGeneratorOption()
}

// Example of creating a hub with AfterSend via functional option
func ExampleWithAfterSendOption() {
	// Define AfterSend behavior via functional option
	afterSendFunc := func(dv *tok.Device, data []byte) {
		slog.Info("AfterSend via functional option", "dv", &dv, "data", string(data))
		// Add custom logic here without implementing full Actor
	}

	config := tok.NewHubConfig(&MinimalActor{},
		tok.WithHubConfigPingProducer(&SimplePingProducer{}),
		tok.WithHubConfigAfterSend(afterSendFunc),
	)

	// Use config to create hub
	_ = config
	slog.Info("Hub configuration created with AfterSend functional option")
}

// CustomCloseHandler demonstrates implementing the CloseHandler interface
type CustomCloseHandler struct{}

func (h *CustomCloseHandler) OnClose(dv *tok.Device) {
	slog.Info("CustomCloseHandler.OnClose", "dv", &dv)
	// Add custom close handling logic here
	// CloseHandler is now the only way to handle close events (Actor.OnClose was removed)
}

// Example of creating a hub with CloseHandler via functional option
func ExampleWithCloseHandlerOption() {
	// Create a custom close handler
	closeHandler := &CustomCloseHandler{}

	config := tok.NewHubConfig(&MinimalActor{},
		tok.WithHubConfigPingProducer(&SimplePingProducer{}),
		tok.WithHubConfigCloseHandler(closeHandler),
	)

	// Use config to create hub
	_ = config
	slog.Info("Hub configuration created with CloseHandler functional option")
}

// Example of creating a hub with ByeGenerator via functional option
func ExampleWithByeGeneratorOption() {
	// Create a bye generator
	byeGenerator := &SimpleByeGenerator{}

	config := tok.NewHubConfig(&MinimalActor{},
		tok.WithHubConfigPingProducer(&SimplePingProducer{}),
		tok.WithHubConfigByeGenerator(byeGenerator),
	)

	// Use config to create hub
	_ = config
	slog.Info("Hub configuration created with ByeGenerator functional option")
}
