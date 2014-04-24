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
	Auth(r *http.Request) (interface{}, error) //auth against http request. return uid if auth success
	OnReceive(uid interface{}, data []byte)    //is invoked every time the server receive valid payload
	//is invoked after a connection has been closed
	//active, count of active connections for this user
	OnClose(uid interface{}, active int)
	Ping() []byte             //Build ping payload.  auto ping feature will be disabled if this method return nil
	Bye(reason string) []byte //Build payload for different reason before connection is closed
}
