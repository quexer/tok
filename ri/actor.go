/**
 * actor reference implementation
 */
package ri

import (
	"code.google.com/p/goprotobuf/proto"
	"fmt"
	"github.com/quexer/kodec"
	"log"
	"github.com/quexer/tok"
	"time"
)

type Checker interface {
	AllowChat(from int, to string) bool
	OnDispatch(interface{})
}

type Actor struct {
	checker Checker
}

func (p *Actor) OnReceive(uid int, data []byte) error {
	m, err := kodec.Unboxing(data)
	if err != nil {
		log.Println("decode err ", err)
		return err
	}

	switch v := m.(type) {
	case *kodec.Msg:
		v.From = proto.Int64(int64(uid))
		v.Ct = proto.Int64(tick())
		//	log.Println("dispatch msg")
		if v.GetTp() != kodec.Msg_SYS {
			//check non-system message
			from := int(v.GetFrom())
			to := v.GetTo()
			if !p.checker.AllowChat(from, to) {
				return fmt.Errorf("warning: chat not allow, %d -> %v \n", from, to)
			}
		}
		p.checker.OnDispatch(v)
		return nil
	default:
		return fmt.Errorf("unknown data frame")
	}
}

func (p *Actor) Ping() []byte {
	b, err := kodec.Boxing(kodec.BuildCmd(kodec.Cmd_PING, "", tick()))
	if err != nil {
		log.Panic("build ping err", err)
	}
	return b
}

func (p *Actor) Bye(reason string) []byte {
	b, err := kodec.Boxing(kodec.BuildCmd(kodec.Cmd_UNAUTHORIZED, reason, tick()))
	if err != nil {
		log.Panic("build bye err", err)
	}
	return b
}

func tick() int64 {
	return time.Now().UnixNano() / 1e6
}

func CreateActor(checker Checker) tok.Actor {
	return &Actor{checker: checker}
}
