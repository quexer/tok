package tok_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/quexer/tok"
)

func TestTok(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tok Suite")
}

var ctx context.Context
var actor tok.Actor
var _ = BeforeEach(func() {
	ctx = context.Background()
	actor = &simpleActor{}
})

type simpleActor struct {
}

func (p *simpleActor) OnReceive(dv *tok.Device, data []byte) {
	return
}

func (p *simpleActor) OnClose(dv *tok.Device) {
	return
}

func (p *simpleActor) Ping() []byte {
	return []byte("pong")
}

func (p *simpleActor) Bye(kicker *tok.Device, reason string, dv *tok.Device) []byte {
	return nil
}

type trackingActor struct {
	AfterSendCalled bool
	AfterSendDevice *tok.Device
	AfterSendData   []byte
}

func (p *trackingActor) OnReceive(dv *tok.Device, data []byte) {
	return
}

func (p *trackingActor) OnClose(dv *tok.Device) {
	return
}

func (p *trackingActor) Ping() []byte {
	return []byte("pong")
}

func (p *trackingActor) Bye(kicker *tok.Device, reason string, dv *tok.Device) []byte {
	return nil
}
