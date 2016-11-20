/**
 * "talk"
 */

package tok

import (
	"errors"
)

//ErrOffline occurs while sending message to online user only. see Hub.Send
var ErrOffline = errors.New("offline")

//ErrQueueRequired occurs while sending "cacheable" message without queue
var ErrQueueRequired = errors.New("queue is required")

//Actor application should implement this interface to interact with tok
type Actor interface {
	Auth(dv Device) (interface{}, error)                  //auth against http request. return uid if auth success
	BeforeReceive(dv Device, data []byte) ([]byte, error) //is invoked before OnReceive
	OnReceive(dv Device, data []byte)                     //is invoked every time the server receive valid payload
	BeforeSend(dv Device, data []byte) ([]byte, error)    //is invoked before Send, if return value is nil, use raw data
	OnSent(dv Device, data []byte)                        //is invoked if message is sent successfully. count mean copy quantity
	//is invoked after a connection has been closed
	//active, count of active connections for this user
	OnClose(dv Device)
	Ping() []byte                        //Build ping payload.  auto ping feature will be disabled if this method return nil
	Bye(dv Device, reason string) []byte //Build payload for different reason before connection is closed
}

const (
	//META_HEADER key for meta data
	META_HEADER = "Tok-Meta"
	//DV_HEADER key for device info
	DV_HEADER = "Tok-Dv"
)
