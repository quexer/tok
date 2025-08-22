package tok_test

import (
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/quexer/tok"
	"github.com/quexer/tok/mocks"
)

var _ = Describe("WsConn", func() {
	var auth tok.WsAuthFunc
	var mActor *mocks.MockActor
	var mPingGen *mocks.MockPingGenerator

	BeforeEach(func() {
		auth = func(r *http.Request) (*tok.Device, error) {
			return tok.CreateDevice(fmt.Sprintf("%p", r), ""), nil
		}
		mActor = mocks.NewMockActor(ctl)
		mPingGen = mocks.NewMockPingGenerator(ctl)
	})

	It("CreateWsHandler", func() {
		hub, hdl := tok.CreateWsHandler(auth,
			tok.WithWsHandlerTxt(true),
			tok.WithWsHandlerHubConfig(tok.NewHubConfig(mActor,
				tok.WithHubConfigSso(true),
				tok.WithHubConfigPingProducer(mPingGen))))
		Ω(hub).ToNot(BeNil())
		Ω(hdl).ToNot(BeNil())
	})

	It("CreateWsHandler with Coder WebSocket", func() {
		hub, hdl := tok.CreateWsHandler(auth,
			tok.WithWsHandlerEngine(tok.WsEngineCoder),
			tok.WithWsHandlerHubConfig(tok.NewHubConfig(mActor,
				tok.WithHubConfigPingProducer(mPingGen))))
		Ω(hub).ToNot(BeNil())
		Ω(hdl).ToNot(BeNil())
	})

	It("CreateWsHandler with Engine enum - XNet", func() {
		hub, hdl := tok.CreateWsHandler(auth,
			tok.WithWsHandlerEngine(tok.WsEngineX),
			tok.WithWsHandlerHubConfig(tok.NewHubConfig(mActor,
				tok.WithHubConfigPingProducer(mPingGen))))
		Ω(hub).ToNot(BeNil())
		Ω(hdl).ToNot(BeNil())
	})

	It("CreateWsHandler with Engine enum - Gorilla", func() {
		hub, hdl := tok.CreateWsHandler(auth,
			tok.WithWsHandlerEngine(tok.WsEngineGorilla),
			tok.WithWsHandlerHubConfig(tok.NewHubConfig(mActor,
				tok.WithHubConfigPingProducer(mPingGen))))
		Ω(hub).ToNot(BeNil())
		Ω(hdl).ToNot(BeNil())
	})
})
