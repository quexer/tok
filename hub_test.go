package tok_test

import (
	"errors"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"github.com/quexer/tok"
	"github.com/quexer/tok/mocks"
)

var _ = Describe("Hub", func() {
	var (
		mockActor *mocks.MockActor
		mockQueue *mocks.MockQueue
		hub       *tok.Hub
		device    *tok.Device
		auth      tok.WsAuthFunc
	)

	BeforeEach(func() {
		mockActor = mocks.NewMockActor(ctl)
		mockQueue = mocks.NewMockQueue(ctl)
		device = tok.CreateDevice("test-user", "test-session")

		// Create hub through public API
		mockPing := mocks.NewMockPingGenerator(ctl)
		mockPing.EXPECT().Ping().Return([]byte("ping")).AnyTimes()

		auth = func(*http.Request) (*tok.Device, error) { return device, nil }
		hubConfig := tok.NewHubConfig(mockActor,
			tok.WithHubConfigQueue(mockQueue),
			tok.WithHubConfigSso(true),
			tok.WithHubConfigPingProducer(mockPing),
		)
		hub, _ = tok.CreateWsHandler(auth, tok.WithWsHandlerHubConfig(hubConfig))

	})

	Describe("Send", func() {
		Context("offline", func() {
			Context("has queue", func() {
				BeforeEach(func() {
					config := tok.NewHubConfig(mockActor, tok.WithHubConfigQueue(mockQueue))
					hub, _ = tok.CreateWsHandler(auth, tok.WithWsHandlerHubConfig(config))
				})

				It("return ErrOffline when TTL is 0", func() {
					err := hub.Send("offline-user", []byte("test"), 0)
					Expect(err).To(Equal(tok.ErrOffline))
				})

				It("cache message when TTL > 0", func() {
					const ttl = 60
					mockQueue.EXPECT().Enq(gomock.Any(), "offline-user", []byte("test"), uint32(ttl))

					err := hub.Send("offline-user", []byte("test"), ttl)
					Ω(err).NotTo(HaveOccurred())
				})

				It("return error when caching fails", func() {
					expectedErr := errors.New("queue error")
					mockQueue.EXPECT().Enq(gomock.Any(), "offline-user", []byte("test"), uint32(60)).
						Return(expectedErr)

					err := hub.Send("offline-user", []byte("test"), 60)
					Ω(err).To(HaveOccurred())
					Ω(err).To(MatchError(expectedErr))
					Ω(err).To(MatchError(tok.ErrCacheFailed))

				})
			})
			Context("queue is nil", func() {
				BeforeEach(func() {
					config := tok.NewHubConfig(mockActor)
					hub, _ = tok.CreateWsHandler(auth, tok.WithWsHandlerHubConfig(config))
				})
				It("should return ErrQueueRequired when trying to cache with TTL > 0", func() {
					err := hub.Send("offline-user", []byte("test"), 60)
					Ω(err).To(HaveOccurred())
					Ω(err).To(MatchError(tok.ErrQueueRequired))
				})
			})
		})
	})

	Describe("CheckOnline method", func() {
		It("should return false for non-existent user", func() {
			online := hub.CheckOnline("non-existent-user")
			Expect(online).To(BeFalse())
		})

		It("should return false for offline user", func() {
			online := hub.CheckOnline("offline-user")
			Expect(online).To(BeFalse())
		})
	})

	Describe("Online method", func() {
		It("should return empty list when no users are online", func() {
			onlineUsers := hub.Online()
			Expect(onlineUsers).To(BeEmpty())
		})
	})

	Describe("Kick method", func() {
		It("should not panic when kicking non-existent user", func() {
			Expect(func() {
				hub.Kick("non-existent-user")
			}).ToNot(Panic())
		})
	})

	Describe("Hub with BeforeReceiveHandler", func() {
		var mockBeforeReceive *mocks.MockBeforeReceiveHandler

		BeforeEach(func() {
			mockBeforeReceive = mocks.NewMockBeforeReceiveHandler(ctl)
			mockPing := mocks.NewMockPingGenerator(ctl)
			mockPing.EXPECT().Ping().Return([]byte("ping")).AnyTimes()

			auth := func(*http.Request) (*tok.Device, error) { return device, nil }
			hubConfig := tok.NewHubConfig(mockActor,
				tok.WithHubConfigBeforeReceive(mockBeforeReceive),
				tok.WithHubConfigPingProducer(mockPing),
			)
			hub, _ = tok.CreateWsHandler(auth, tok.WithWsHandlerHubConfig(hubConfig))
		})

		It("should work with BeforeReceive handler configured", func() {
			// This test verifies the hub can be created with BeforeReceiveHandler
			// The actual handler behavior would be tested in integration tests
			Expect(hub).ToNot(BeNil())
		})
	})

	Describe("Hub with BeforeSendHandler", func() {
		var mockBeforeSend *mocks.MockBeforeSendHandler

		BeforeEach(func() {
			mockBeforeSend = mocks.NewMockBeforeSendHandler(ctl)
			mockPing := mocks.NewMockPingGenerator(ctl)
			mockPing.EXPECT().Ping().Return([]byte("ping")).AnyTimes()

			auth := func(*http.Request) (*tok.Device, error) { return device, nil }
			hubConfig := tok.NewHubConfig(mockActor,
				tok.WithHubConfigBeforeSend(mockBeforeSend),
				tok.WithHubConfigPingProducer(mockPing),
			)
			hub, _ = tok.CreateWsHandler(auth, tok.WithWsHandlerHubConfig(hubConfig))
		})

		It("should work with BeforeSend handler configured", func() {
			// This test verifies the hub can be created with BeforeSendHandler
			// The actual handler behavior would be tested in integration tests
			Expect(hub).ToNot(BeNil())
		})
	})

	Describe("Hub with AfterSendHandler", func() {
		var mockAfterSend *mocks.MockAfterSendHandler

		BeforeEach(func() {
			mockAfterSend = mocks.NewMockAfterSendHandler(ctl)
			mockPing := mocks.NewMockPingGenerator(ctl)
			mockPing.EXPECT().Ping().Return([]byte("ping")).AnyTimes()

			auth := func(*http.Request) (*tok.Device, error) { return device, nil }
			hubConfig := tok.NewHubConfig(mockActor,
				tok.WithHubConfigAfterSend(mockAfterSend),
				tok.WithHubConfigPingProducer(mockPing),
			)
			hub, _ = tok.CreateWsHandler(auth, tok.WithWsHandlerHubConfig(hubConfig))
		})

		It("should work with AfterSend handler configured", func() {
			// This test verifies the hub can be created with AfterSendHandler
			// The actual handler behavior would be tested in integration tests
			Expect(hub).ToNot(BeNil())
		})
	})

	Describe("Hub with CloseHandler", func() {
		var mockCloseHandler *mocks.MockCloseHandler

		BeforeEach(func() {
			mockCloseHandler = mocks.NewMockCloseHandler(ctl)
			mockPing := mocks.NewMockPingGenerator(ctl)
			mockPing.EXPECT().Ping().Return([]byte("ping")).AnyTimes()

			auth := func(*http.Request) (*tok.Device, error) { return device, nil }
			hubConfig := tok.NewHubConfig(mockActor,
				tok.WithHubConfigCloseHandler(mockCloseHandler),
				tok.WithHubConfigPingProducer(mockPing),
			)
			hub, _ = tok.CreateWsHandler(auth, tok.WithWsHandlerHubConfig(hubConfig))
		})

		It("should work with CloseHandler configured", func() {
			// This test verifies the hub can be created with CloseHandler
			// The actual handler behavior would be tested in integration tests
			Expect(hub).ToNot(BeNil())
		})
	})

	Describe("Hub with PingGenerator", func() {
		var mockPingGenerator *mocks.MockPingGenerator

		BeforeEach(func() {
			mockPingGenerator = mocks.NewMockPingGenerator(ctl)
			auth := func(*http.Request) (*tok.Device, error) { return device, nil }
			hubConfig := tok.NewHubConfig(mockActor,
				tok.WithHubConfigPingProducer(mockPingGenerator),
			)
			hub, _ = tok.CreateWsHandler(auth, tok.WithWsHandlerHubConfig(hubConfig))
		})

		It("should work with PingGenerator configured", func() {
			// This test verifies the hub can be created with PingGenerator
			// The actual ping behavior would be tested in integration tests
			Expect(hub).ToNot(BeNil())
		})
	})

	Describe("Hub queue operations", func() {
		Context("with valid queue", func() {
			It("should handle queue operations for offline users", func() {
				// Test caching to queue for offline user
				mockQueue.EXPECT().Enq(gomock.Any(), "user123", []byte("message"), uint32(300)).Return(nil).AnyTimes()

				err := hub.Send("user123", []byte("message"), 300)
				Expect(err).To(BeNil())
			})

			It("should handle queue errors gracefully", func() {
				queueErr := errors.New("queue full")
				mockQueue.EXPECT().Enq(gomock.Any(), "user123", []byte("message"), uint32(300)).Return(queueErr).AnyTimes()

				err := hub.Send("user123", []byte("message"), 300)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, queueErr)).To(BeTrue())
			})
		})
	})

	Describe("Hub configuration variations", func() {
		It("should work with SSO disabled", func() {
			mockPing := mocks.NewMockPingGenerator(ctl)
			mockPing.EXPECT().Ping().Return([]byte("ping")).AnyTimes()

			auth := func(*http.Request) (*tok.Device, error) { return device, nil }
			hubConfig := tok.NewHubConfig(mockActor,
				tok.WithHubConfigSso(false),
				tok.WithHubConfigQueue(mockQueue),
				tok.WithHubConfigPingProducer(mockPing),
			)
			hub, _ := tok.CreateWsHandler(auth, tok.WithWsHandlerHubConfig(hubConfig))

			Expect(hub).ToNot(BeNil())
			Expect(hub.CheckOnline("non-existent")).To(BeFalse())
		})

		It("should work with multiple handlers configured", func() {
			mockBeforeReceive := mocks.NewMockBeforeReceiveHandler(ctl)
			mockBeforeSend := mocks.NewMockBeforeSendHandler(ctl)
			mockAfterSend := mocks.NewMockAfterSendHandler(ctl)
			mockCloseHandler := mocks.NewMockCloseHandler(ctl)
			mockPingGenerator := mocks.NewMockPingGenerator(ctl)

			auth := func(*http.Request) (*tok.Device, error) { return device, nil }
			hubConfig := tok.NewHubConfig(mockActor,
				tok.WithHubConfigQueue(mockQueue),
				tok.WithHubConfigSso(true),
				tok.WithHubConfigBeforeReceive(mockBeforeReceive),
				tok.WithHubConfigBeforeSend(mockBeforeSend),
				tok.WithHubConfigAfterSend(mockAfterSend),
				tok.WithHubConfigCloseHandler(mockCloseHandler),
				tok.WithHubConfigPingProducer(mockPingGenerator),
			)
			hub, _ := tok.CreateWsHandler(auth, tok.WithWsHandlerHubConfig(hubConfig))

			Expect(hub).ToNot(BeNil())
		})
	})
})
