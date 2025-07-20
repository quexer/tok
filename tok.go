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

// BeforeReceiveHandler is an interface for preprocessing incoming data before OnReceive
type BeforeReceiveHandler interface {
	// BeforeReceive is called to preprocess incoming data before OnReceive
	BeforeReceive(dv *Device, data []byte) ([]byte, error)
}

// BeforeSendHandler is an interface for preprocessing outgoing data before sending
type BeforeSendHandler interface {
	// BeforeSend is called to preprocess outgoing data before sending
	BeforeSend(dv *Device, data []byte) ([]byte, error)
}

// AfterSendHandler is an interface for handling events after sending data
type AfterSendHandler interface {
	// AfterSend is called after data has been sent to a device
	AfterSend(dv *Device, data []byte)
}

// CloseHandler is an interface for handling connection close events
type CloseHandler interface {
	// OnClose is called after a connection has been closed
	OnClose(dv *Device)
}

// PingGenerator is an interface for generating server-side ping payloads
type PingGenerator interface {
	// Ping generate server-side ping payload
	Ping() []byte
}

// ByeGenerator is an interface for generating bye payloads
type ByeGenerator interface {
	// Bye builds the payload to notify before a connection is closed for a specific reason.
	Bye(kicker *Device, reason string, dv *Device) []byte
}

// Actor should be implemented by applications to interact with tok.
// Each method provides a hook for handling device communication events.
type Actor interface {
	// OnReceive is called whenever the server receives a valid payload.
	OnReceive(dv *Device, data []byte)
}
