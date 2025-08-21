package tok_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"github.com/quexer/tok"
)

func TestTok(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tok Suite")
}

var ctx context.Context
var ctl *gomock.Controller
var actor tok.Actor
var _ = BeforeEach(func() {
	ctx = context.Background()
	ctl = gomock.NewController(GinkgoT())
	actor = &simpleActor{}
})

var _ = AfterEach(func() {
	ctl.Finish()
})

type simpleActor struct {
}

func (p *simpleActor) OnReceive(dv *tok.Device, data []byte) {
	return
}

type trackingActor struct {
	AfterSendCalled bool
	AfterSendDevice *tok.Device
	AfterSendData   []byte
}

func (p *trackingActor) OnReceive(dv *tok.Device, data []byte) {
	return
}

// Test PingGenerator implementations
type testPingGenerator struct{}

func (p *testPingGenerator) Ping() []byte {
	return []byte("ping")
}

// Test ByeGenerator implementations
type testByeGenerator struct{}

func (b *testByeGenerator) Bye(kicker *tok.Device, reason string, dv *tok.Device) []byte {
	return []byte("bye")
}

// Test BeforeSendHandler implementation
type testBeforeSendHandler struct {
	transform func(dv *tok.Device, data []byte) ([]byte, error)
}

func (h *testBeforeSendHandler) BeforeSend(dv *tok.Device, data []byte) ([]byte, error) {
	if h.transform != nil {
		return h.transform(dv, data)
	}
	return data, nil
}

// Test AfterSendHandler implementation
type testAfterSendHandler struct {
	afterSendCalled bool
	afterSendDevice *tok.Device
	afterSendData   []byte
	callback        func(dv *tok.Device, data []byte)
}

func (h *testAfterSendHandler) AfterSend(dv *tok.Device, data []byte) {
	h.afterSendCalled = true
	h.afterSendDevice = dv
	h.afterSendData = data
	if h.callback != nil {
		h.callback(dv, data)
	}
}
