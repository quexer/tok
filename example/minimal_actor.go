package main

import (
	"fmt"
	"log/slog"

	"github.com/quexer/tok"
)

// MinimalActor demonstrates an actor that only implements the required methods
// OnSent can be provided via functional options instead
type MinimalActor struct{}

func (a *MinimalActor) OnReceive(dv *tok.Device, data []byte) {
	slog.Info("MinimalActor.OnReceive", "dv", &dv, "data", string(data))
}

// OnSent is still required by the Actor interface for backward compatibility
// but can be empty if using the functional option
func (a *MinimalActor) OnSent(dv *tok.Device, data []byte) {
	// This will not be called if WithHubConfigOnSent is provided
	slog.Info("MinimalActor.OnSent (fallback)", "dv", &dv, "data", string(data))
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
	// Example of creating a hub with OnSent via functional option
	ExampleWithOnSentOption()
}

// Example of creating a hub with OnSent via functional option
func ExampleWithOnSentOption() {
	// Define OnSent behavior via functional option
	onSentFunc := func(dv *tok.Device, data []byte) {
		slog.Info("OnSent via functional option", "dv", &dv, "data", string(data))
		// Add custom logic here without implementing full Actor
	}

	config := tok.NewHubConfig(&MinimalActor{},
		tok.WithHubConfigOnSent(onSentFunc),
	)

	// Use config to create hub
	_ = config
	slog.Info("Hub configuration created with OnSent functional option")
}