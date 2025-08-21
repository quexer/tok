/**
 * "talk"
 */

package tok

import (
	"errors"
)

//go:generate mockgen -destination=mocks/tok.go -package=mocks . Actor,BeforeReceiveHandler,BeforeSendHandler,AfterSendHandler,CloseHandler,PingGenerator

// ErrOffline occurs while sending message to online user only. see Hub.Send
var ErrOffline = errors.New("tok: offline")

// ErrQueueRequired occurs while sending "cacheable" message without queue
var ErrQueueRequired = errors.New("tok: queue is required")

// ErrCacheFailed occurs while sending "cacheable" message with queue but failed to cache
var ErrCacheFailed = errors.New("tok: cache error")

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
	// Bye builds the payload to notify clients before a connection is closed for a specific reason.
	// kicker is the device that initiated the kick, reason is the reason for the kick, dv is the device being kicked.
	Bye(kicker *Device, reason string, dv *Device) []byte
}

// Actor interface is used to handle valid payloads received by the server.
type Actor interface {
	// OnReceive is called whenever the server receives a valid payload.
	// dv represents the sender the data, data is the received byte slice.
	OnReceive(dv *Device, data []byte)
}
