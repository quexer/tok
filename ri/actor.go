/**
 * actor reference implementation
 */
package ri

import (
	"github.com/golang/protobuf/proto"
	"fmt"
	"github.com/quexer/kodec"
	"github.com/quexer/tok"
	"log"
	"net/http"
	"time"
)

type Checker interface {
	CheckUp(from interface{}, to string) bool
	CheckDown(target interface{}, v *kodec.Msg) bool
	Dispatch(targets []interface{}, v *kodec.Msg)
	ParseAddr(to string) ([]interface{}, error)
	Cached(uid interface {})
	Auth(r *http.Request) (interface{}, error)
	BeforeReceive(uid interface{}, data []byte) []byte
	BeforeSend(uid interface{}, data []byte) []byte
}

type Actor struct {
	checker Checker
}

func (p *Actor) BeforeReceive(uid interface{}, data []byte) []byte{
	return p.checker.BeforeReceive(uid, data)
}
func (p *Actor) BeforeSend(uid interface{}, data []byte) []byte{
	return p.checker.BeforeSend(uid, data)
}

func (p *Actor) OnReceive(uid interface{}, data []byte) {
	m, err := kodec.Unboxing(data)
	if err != nil {
		log.Println("decode err ", err)
		return
	}

	switch v := m.(type) {
	case *kodec.Msg:
		v.From = proto.Int64(int64(uid.(int)))
		v.Ct = proto.Int64(tick())

		if v.GetTp() != kodec.Msg_SYS {
			from := int(v.GetFrom())
			to := v.GetTo()
			if !p.checker.CheckUp(from, to) {
				log.Println(fmt.Errorf("warning: chat not allow, %d -> %v \n", from, to))
				return
			}
		}
		p.dispatchMsg(v)
	default:
		log.Println(fmt.Errorf("unknown data frame"))
	}
}

func (p *Actor) dispatchMsg(v *kodec.Msg) {
	uids := []interface{}{}
	targets, err := p.checker.ParseAddr(v.GetTo())
	if err != nil {
		log.Println("parse addr err", err)
		return
	}

	from := int(v.GetFrom())
	for _, target := range targets {
		if target == from || !p.checker.CheckDown(target, v) {
			continue
		}
		uids = append(uids, target)
	}

	go p.checker.Dispatch(uids, v)
}

func (p *Actor) OnSent(uid interface{}, data []byte, count int) {
	//do nothing
}

func (p *Actor) OnCache(uid interface{}) {
	p.checker.Cached(uid)
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

func (p *Actor) Auth(r *http.Request) (interface{}, error) {
	return p.checker.Auth(r)
}

func (p *Actor) OnClose(uid interface{}, active int) {
	//do nothing
}

func tick() int64 {
	return time.Now().UnixNano() / 1e6
}

func CreateActor(checker Checker) tok.Actor {
	return &Actor{checker: checker}
}
