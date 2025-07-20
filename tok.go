/**
 * "talk"
 */

package tok

import (
	"errors"
)

// ErrOffline occurs while sending message to online user only. see Hub.Send
var ErrOffline = errors.New("offline")

// ErrQueueRequired occurs while sending "cacheable" message without queue
var ErrQueueRequired = errors.New("queue is required")

// BeforeReceiveFunc is a function type for preprocessing incoming data before OnReceive
type BeforeReceiveFunc func(dv *Device, data []byte) ([]byte, error)

// BeforeSendFunc is a function type for preprocessing outgoing data before sending
type BeforeSendFunc func(dv *Device, data []byte) ([]byte, error)

// Actor should be implemented by applications to interact with tok.
// Each method provides a hook for handling device communication events.
type Actor interface {
	// OnReceive is called whenever the server receives a valid payload.
	OnReceive(dv *Device, data []byte)
	// OnClose is called after a connection has been closed.
	OnClose(dv *Device)
	// Ping builds the ping payload. If nil is returned, the auto-ping feature is disabled.
	Ping() []byte
	// Bye builds the payload to notify before a connection is closed for a specific reason.
	Bye(kicker *Device, reason string, dv *Device) []byte
}
