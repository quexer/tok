/**
 * actor reference implementation
 */
package ri

import (
	"code.google.com/p/goprotobuf/proto"
	"fmt"
	"github.com/quexer/kodec"
	"github.com/quexer/tok"
	"log"
	"time"
)

type Checker interface {
	CheckUp(from int, to string) bool
	CheckDown(target int, v *kodec.Msg) bool
	PostDispatch(target int, v *kodec.Msg)
	ParseAddr(to string) ([]int, error)
	UserStateChange(target int, online bool)
}

type Actor struct {
	checker Checker
}

func (p *Actor) OnReceive(uid int, data []byte) ([]int, error) {
	m, err := kodec.Unboxing(data)
	if err != nil {
		log.Println("decode err ", err)
		return nil, err
	}

	switch v := m.(type) {
	case *kodec.Msg:
		v.From = proto.Int64(int64(uid))
		v.Ct = proto.Int64(tick())
		if v.GetTp() != kodec.Msg_SYS {
			from := int(v.GetFrom())
			to := v.GetTo()
			if !p.checker.CheckUp(from, to) {
				return nil, fmt.Errorf("warning: chat not allow, %d -> %v \n", from, to)
			}
		}
		return p.dispatchMsg(v)
	default:
		return nil, fmt.Errorf("unknown data frame")
	}
}

func (p *Actor) dispatchMsg(v *kodec.Msg) ([]int, error) {
	uids := []int{}
	targets, err := p.checker.ParseAddr(v.GetTo())
	if err != nil {
		log.Println("parse addr err", err)
		return nil, err
	}

	from := int(v.GetFrom())
	for _, target := range targets {
		if target == from || !p.checker.CheckDown(target, v) {
			continue
		}

		uids = append(uids, target)
		go p.checker.PostDispatch(target, v)
	}

	return uids, nil
}

func (p *Actor) OnChange(uid int, online bool){
	p.checker.UserStateChange(uid, online)
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
