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
	CheckUp(from interface{}, to string) bool
	CheckDown(target interface{}, v *kodec.Msg) bool
	PostDispatch(target interface{}, v *kodec.Msg)
	ParseAddr(to string) ([]interface{}, error)
}

type Actor struct {
	checker Checker
}

func (p *Actor) OnReceive(uid interface {}, data []byte) ([]interface {}, []byte, error) {
	m, err := kodec.Unboxing(data)
	if err != nil {
		log.Println("decode err ", err)
		return nil, nil, err
	}

	switch v := m.(type) {
	case *kodec.Msg:
		v.From = proto.Int64(int64(uid.(int)))
		v.Ct = proto.Int64(tick())

		b, err := kodec.Boxing(v)
		if err != nil {
			log.Println("build replay binary err", err)
			return nil, nil, err
		}

		if v.GetTp() != kodec.Msg_SYS {
			from := int(v.GetFrom())
			to := v.GetTo()
			if !p.checker.CheckUp(from, to) {
				return nil, nil, fmt.Errorf("warning: chat not allow, %d -> %v \n", from, to)
			}
		}
		target, err := p.dispatchMsg(v)
		if err != nil {
			log.Println("dispatch err", err)
			return nil, nil, err
		}
		return target, b, err
	default:
		return nil, nil, fmt.Errorf("unknown data frame")
	}
}

func (p *Actor) dispatchMsg(v *kodec.Msg) ([]interface {}, error) {
	uids := []interface{}{}
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
