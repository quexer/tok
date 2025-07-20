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

func (a *MinimalActor) OnClose(dv *tok.Device) {
	slog.Info("MinimalActor.OnClose", "dv", &dv)
}

func (a *MinimalActor) Ping() []byte {
	return []byte("ping")
}

func (a *MinimalActor) Bye(kicker *tok.Device, reason string, dv *tok.Device) []byte {
	return []byte(fmt.Sprintf("bye: %s", reason))
}

func main() {
	// Example of creating a hub with AfterSend via functional option
	ExampleWithAfterSendOption()
}

// Example of creating a hub with AfterSend via functional option
func ExampleWithAfterSendOption() {
	// Define AfterSend behavior via functional option
	afterSendFunc := func(dv *tok.Device, data []byte) {
		slog.Info("AfterSend via functional option", "dv", &dv, "data", string(data))
		// Add custom logic here without implementing full Actor
	}

	config := tok.NewHubConfig(&MinimalActor{},
		tok.WithHubConfigAfterSend(afterSendFunc),
	)

	// Use config to create hub
	_ = config
	slog.Info("Hub configuration created with AfterSend functional option")
}