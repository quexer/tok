/**
 * "talk"
 */

package tok

import (
	"errors"
	"net/http"
)

var ErrOffline = errors.New("offline")
var ErrQueueRequired = errors.New("queue is required")

//Application can interact with tok via this interface
type Actor interface {
	Auth(r *http.Request) (interface{}, error)         //auth against http request. return uid if auth success
	BeforeReceive(uid interface{}, data []byte) []byte //is invoked before OnReceive
	OnReceive(uid interface{}, data []byte)            //is invoked every time the server receive valid payload
	BeforeSend(uid interface{}, data []byte) []byte    //is invoked before Send
	OnSent(uid interface{}, data []byte, count int)    //is invoked if message is sent successfully. count mean copy quantity
	OnCache(uid interface{})                           //is invoked after message caching
	//is invoked after a connection has been closed
	//active, count of active connections for this user
	OnClose(uid interface{}, active int)
	Ping() []byte             //Build ping payload.  auto ping feature will be disabled if this method return nil
	Bye(uid interface{}, reason string) []byte //Build payload for different reason before connection is closed
}

const (
	META_HEADER = "Tok-Meta"
	DV_HEADER   = "Tok-Dv"
)
