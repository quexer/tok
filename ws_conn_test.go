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

	It("CreateWsHandler with default settings", func() {
		hub, hdl, err := tok.CreateWsHandler(ctx, auth,
			tok.WithWsHandlerHubConfig(tok.NewHubConfig(mActor,
				tok.WithHubConfigPingProducer(mPingGen))))
		Ω(err).To(Succeed())
		Ω(hub).ToNot(BeNil())
		Ω(hdl).ToNot(BeNil())
	})

	DescribeTable("CreateWsHandler with WithWsHandlerHub (no hubConfig) should not panic",
		func(engine tok.WsEngine) {
			// Create hub via a normal path first
			hub, _, err := tok.CreateWsHandler(ctx, auth,
				tok.WithWsHandlerHubConfig(tok.NewHubConfig(mActor,
					tok.WithHubConfigPingProducer(mPingGen))))
			Ω(err).ToNot(HaveOccurred())
			defer hub.Close()

			// Now use WithWsHandlerHub with nil hubConfig — this used to panic
			_, hdl, err := tok.CreateWsHandler(ctx, auth,
				tok.WithWsHandlerHub(hub),
				tok.WithWsHandlerEngine(engine))
			Ω(err).To(Succeed())
			Ω(hdl).ToNot(BeNil())
		},
		Entry("X WebSocket", tok.WsEngineX),
		Entry("Gorilla WebSocket", tok.WsEngineGorilla),
		Entry("Coder WebSocket", tok.WsEngineCoder),
	)

	DescribeTable("CreateWsHandler with different WebSocket engines",
		func(engine tok.WsEngine) {
			hub, hdl, err := tok.CreateWsHandler(ctx, auth,
				tok.WithWsHandlerEngine(engine),
				tok.WithWsHandlerHubConfig(tok.NewHubConfig(mActor,
					tok.WithHubConfigPingProducer(mPingGen))))
			Ω(err).To(Succeed())
			Ω(hub).ToNot(BeNil())
			Ω(hdl).ToNot(BeNil())
		},
		Entry("Coder WebSocket", tok.WsEngineCoder),
		Entry("XNet WebSocket", tok.WsEngineX),
		Entry("Gorilla WebSocket", tok.WsEngineGorilla),
	)
})
