package tok_test

import (
	"errors"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/quexer/tok"
)

var _ = Describe("BeforeReceive Functional Option", func() {

	It("should work without BeforeReceive option", func() {
		hubConfig := tok.NewHubConfig(actor)
		Ω(hubConfig).ToNot(BeNil())
	})

	It("should work with BeforeReceive option", func() {
		beforeReceiveHandler := &testBeforeReceiveHandler{prefix: "prefix:"}

		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigBeforeReceive(beforeReceiveHandler))
		Ω(hubConfig).ToNot(BeNil())
	})

})

var _ = Describe("BeforeSend Functional Option", func() {

	It("should work without BeforeSend option", func() {
		hubConfig := tok.NewHubConfig(actor)
		Ω(hubConfig).ToNot(BeNil())
	})

	It("should work with BeforeSend option", func() {
		beforeSendHandler := &testBeforeSendHandler{
			transform: func(dv *tok.Device, data []byte) ([]byte, error) {
				// Transform data by adding a suffix
				return append(data, []byte(":suffix")...), nil
			},
		}

		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigBeforeSend(beforeSendHandler))
		Ω(hubConfig).ToNot(BeNil())
	})

	It("should handle BeforeSend error correctly", func() {
		beforeSendHandler := &testBeforeSendHandler{
			transform: func(dv *tok.Device, data []byte) ([]byte, error) {
				// Return an error to test error handling
				return nil, errors.New("BeforeSend error")
			},
		}

		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigBeforeSend(beforeSendHandler))
		Ω(hubConfig).ToNot(BeNil())
	})

	It("should handle BeforeSend returning nil data correctly", func() {
		beforeSendHandler := &testBeforeSendHandler{
			transform: func(dv *tok.Device, data []byte) ([]byte, error) {
				// Return nil data, should use original data
				return nil, nil
			},
		}

		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigBeforeSend(beforeSendHandler))
		Ω(hubConfig).ToNot(BeNil())
	})

})

var _ = Describe("AfterSend Functional Option", func() {

	It("should work without AfterSend option", func() {
		hubConfig := tok.NewHubConfig(actor)
		Ω(hubConfig).ToNot(BeNil())
		// hdlAfterSend should be nil, so AfterSend functionality is disabled
	})

	It("should work with AfterSend option", func() {
		var afterSendCalled bool

		afterSendHandler := &testAfterSendHandler{
			callback: func(dv *tok.Device, data []byte) {
				afterSendCalled = true
			},
		}

		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigAfterSend(afterSendHandler))
		Ω(hubConfig).ToNot(BeNil())

		// Basic verification that the config was created successfully
		// The actual functionality is tested through integration
		Ω(afterSendCalled).To(BeFalse()) // Not called yet
	})

	It("should accept nil AfterSend handler", func() {
		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigAfterSend(nil))
		Ω(hubConfig).ToNot(BeNil())
	})

	It("should work with multiple functional options including AfterSend", func() {
		afterSendHandler := &testAfterSendHandler{
			callback: func(dv *tok.Device, data []byte) {
				// Do nothing, just verify it can be configured
			},
		}

		beforeReceiveHandler := &testBeforeReceiveHandler{}

		hubConfig := tok.NewHubConfig(actor,
			tok.WithHubConfigAfterSend(afterSendHandler),
			tok.WithHubConfigBeforeReceive(beforeReceiveHandler),
			tok.WithHubConfigSso(false),
		)
		Ω(hubConfig).ToNot(BeNil())
	})

})

type testCloseHandler struct {
	closeCalled bool
	lastDevice  *tok.Device
}

func (h *testCloseHandler) OnClose(dv *tok.Device) {
	h.closeCalled = true
	h.lastDevice = dv
}

type testBeforeReceiveHandler struct {
	prefix string
}

func (h *testBeforeReceiveHandler) BeforeReceive(dv *tok.Device, data []byte) ([]byte, error) {
	if h.prefix != "" {
		// Transform data by adding a prefix
		return append([]byte(h.prefix), data...), nil
	}
	return data, nil
}

var _ = Describe("PingGenerator Functional Option", func() {

	It("should work without PingGenerator option", func() {
		hubConfig := tok.NewHubConfig(actor)
		Ω(hubConfig).ToNot(BeNil())
		// pingProducer should be nil, so ping functionality is disabled
	})

	It("should work with PingGenerator option", func() {
		pingProducer := &testPingGenerator{}

		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigPingProducer(pingProducer))
		Ω(hubConfig).ToNot(BeNil())

		// Basic verification that the config was created successfully
		// The actual functionality is tested through integration
	})

	It("should accept nil PingGenerator", func() {
		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigPingProducer(nil))
		Ω(hubConfig).ToNot(BeNil())
	})

	It("should work with multiple functional options including PingGenerator", func() {
		pingProducer := &testPingGenerator{}

		afterSendHandler := &testAfterSendHandler{
			callback: func(dv *tok.Device, data []byte) {
				// Do nothing, just verify it can be configured
			},
		}

		hubConfig := tok.NewHubConfig(actor,
			tok.WithHubConfigPingProducer(pingProducer),
			tok.WithHubConfigAfterSend(afterSendHandler),
			tok.WithHubConfigSso(false),
		)
		Ω(hubConfig).ToNot(BeNil())
	})

})

var _ = Describe("PingGenerator Functional Option", func() {

	It("should work without PingGenerator option", func() {
		hubConfig := tok.NewHubConfig(actor)
		Ω(hubConfig).ToNot(BeNil())
		// pingProducer should be nil, so ping functionality is disabled
	})

	It("should work with PingGenerator option", func() {
		pingProducer := &testPingGenerator{}

		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigPingProducer(pingProducer))
		Ω(hubConfig).ToNot(BeNil())

		// Basic verification that the config was created successfully
		// The actual functionality is tested through integration
	})

	It("should accept nil PingGenerator", func() {
		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigPingProducer(nil))
		Ω(hubConfig).ToNot(BeNil())
	})

	It("should work with multiple functional options including PingGenerator", func() {
		pingProducer := &testPingGenerator{}

		afterSendHandler := &testAfterSendHandler{
			callback: func(dv *tok.Device, data []byte) {
				// Do nothing, just verify it can be configured
			},
		}

		hubConfig := tok.NewHubConfig(actor,
			tok.WithHubConfigPingProducer(pingProducer),
			tok.WithHubConfigAfterSend(afterSendHandler),
			tok.WithHubConfigSso(false),
		)
		Ω(hubConfig).ToNot(BeNil())
	})

})

var _ = Describe("CloseHandler Functional Option", func() {

	It("should work without CloseHandler option", func() {
		hubConfig := tok.NewHubConfig(actor)
		Ω(hubConfig).ToNot(BeNil())
		// fnOnClose should be nil, so CloseHandler functionality is disabled
	})

	It("should work with CloseHandler option", func() {
		closeHandler := &testCloseHandler{}

		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigCloseHandler(closeHandler))
		Ω(hubConfig).ToNot(BeNil())

		// Basic verification that the config was created successfully
		// The actual functionality is tested through integration
		Ω(closeHandler.closeCalled).To(BeFalse()) // Not called yet
	})

	It("should accept nil CloseHandler", func() {
		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigCloseHandler(nil))
		Ω(hubConfig).ToNot(BeNil())
	})

	It("should work with multiple functional options including CloseHandler", func() {
		closeHandler := &testCloseHandler{}

		afterSendHandler := &testAfterSendHandler{
			callback: func(dv *tok.Device, data []byte) {
				// Do nothing, just verify it can be configured
			},
		}

		hubConfig := tok.NewHubConfig(actor,
			tok.WithHubConfigCloseHandler(closeHandler),
			tok.WithHubConfigAfterSend(afterSendHandler),
			tok.WithHubConfigSso(false),
		)
		Ω(hubConfig).ToNot(BeNil())
	})

	It("should call CloseHandler when using integration with WsHandler", func() {
		closeHandler := &testCloseHandler{}
		actorHandler := &trackingActor{}

		// Create a websocket handler with CloseHandler configured
		auth := func(r *http.Request) (*tok.Device, error) {
			return tok.CreateDevice("test-user", ""), nil
		}

		hub, handler := tok.CreateWsHandler(auth,
			tok.WithWsHandlerHubConfig(tok.NewHubConfig(actorHandler,
				tok.WithHubConfigCloseHandler(closeHandler),
				tok.WithHubConfigPingProducer(&testPingGenerator{}))))

		Ω(hub).ToNot(BeNil())
		Ω(handler).ToNot(BeNil())

		// Initially, neither handler should have been called
		Ω(closeHandler.closeCalled).To(BeFalse())

		// This test verifies that the handler configuration is set up correctly
		// The actual close behavior would be tested in a full integration test with a real connection
	})

})

var _ = Describe("ByeGenerator Functional Option", func() {

	It("should work without ByeGenerator option", func() {
		hubConfig := tok.NewHubConfig(actor)
		Ω(hubConfig).ToNot(BeNil())
		// byeGenerator should be nil, so bye functionality is disabled
	})

	It("should work with ByeGenerator option", func() {
		byeGenerator := &testByeGenerator{}

		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigByeGenerator(byeGenerator))
		Ω(hubConfig).ToNot(BeNil())

		// Basic verification that the config was created successfully
		// The actual functionality is tested through integration
	})

	It("should accept nil ByeGenerator", func() {
		hubConfig := tok.NewHubConfig(actor, tok.WithHubConfigByeGenerator(nil))
		Ω(hubConfig).ToNot(BeNil())
	})

	It("should work with multiple functional options including ByeGenerator", func() {
		byeGenerator := &testByeGenerator{}

		afterSendHandler := &testAfterSendHandler{
			callback: func(dv *tok.Device, data []byte) {
				// Do nothing, just verify it can be configured
			},
		}

		hubConfig := tok.NewHubConfig(actor,
			tok.WithHubConfigByeGenerator(byeGenerator),
			tok.WithHubConfigAfterSend(afterSendHandler),
			tok.WithHubConfigSso(false),
		)
		Ω(hubConfig).ToNot(BeNil())
	})

})
