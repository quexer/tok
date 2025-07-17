// This is a demonstration of the BeforeSend functional option
// Before: BeforeSend was a required method in the Actor interface  
// After: BeforeSend is an optional function that can be provided via WithHubConfigBeforeSend

package main

import (
	"fmt"
	"log/slog"

	"github.com/quexer/tok"
)

type myActor struct{}

// Notice: BeforeSend method is NO LONGER required in Actor implementations
func (a *myActor) OnReceive(dv *tok.Device, data []byte) {}
func (a *myActor) OnSent(dv *tok.Device, data []byte)    {}
func (a *myActor) OnClose(dv *tok.Device)                {}
func (a *myActor) Ping() []byte                          { return []byte("ping") }
func (a *myActor) Bye(kicker *tok.Device, reason string, dv *tok.Device) []byte { return nil }

func main() {
	// Example 1: Hub without BeforeSend (now optional)
	fmt.Println("Example 1: Creating hub WITHOUT BeforeSend")
	hubConfig1 := tok.NewHubConfig(&myActor{})
	fmt.Printf("✓ Hub created successfully without BeforeSend: %v\n", hubConfig1 != nil)

	// Example 2: Hub with BeforeSend using functional option
	fmt.Println("\nExample 2: Creating hub WITH BeforeSend using functional option")
	beforeSendFunc := func(dv *tok.Device, data []byte) ([]byte, error) {
		slog.Info("BeforeSend called", "device", dv.UID(), "originalData", string(data))
		// Transform data by adding a prefix
		return append([]byte("TRANSFORMED:"), data...), nil
	}

	hubConfig2 := tok.NewHubConfig(&myActor{},
		tok.WithHubConfigBeforeSend(beforeSendFunc))
	fmt.Printf("✓ Hub created successfully with BeforeSend: %v\n", hubConfig2 != nil)

	fmt.Println("\n✅ BeforeSend is now optional using functional options pattern!")
}