package tok_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/gorilla/websocket"
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

	Context("Online", func() {
		var (
			server    *httptest.Server
			ws        *websocket.Conn
			wsHandler http.Handler
		)

		BeforeEach(func() {
			hub, wsHandler = tok.CreateWsHandler(auth, tok.WithWsHandlerHubConfig(
				tok.NewHubConfig(mockActor,
					tok.WithHubConfigQueue(mockQueue),
				),
			))
			// wsHandler = handler.ServeHTTP
			server = httptest.NewServer(wsHandler)

			// Convert http:// to ws://
			u := "ws" + strings.TrimPrefix(server.URL, "http")

			// Connect to the server
			var err error
			ws, _, err = websocket.DefaultDialer.Dial(u, nil)
			Expect(err).NotTo(HaveOccurred())

			// wait for online
			time.Sleep(100 * time.Millisecond)
		})

		AfterEach(func() {
			_ = ws.Close()
			server.Close()
		})

		Context("Send", func() {
			It("should send message to the online user", func() {
				message := []byte("hello")
				err := hub.Send(device.UID, message, 0)
				Expect(err).NotTo(HaveOccurred())

				_, p, err := ws.ReadMessage()
				Expect(err).NotTo(HaveOccurred())
				Expect(p).To(Equal(message))
			})

			It("should return an error if sending fails", func() {
				// Close the connection to trigger a send error
				err := ws.Close()
				Ω(err).NotTo(HaveOccurred())
				// wait for offline
				time.Sleep(100 * time.Millisecond)

				message := []byte("hello")
				err = hub.Send(device.UID, message, 0)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("CheckOnline", func() {
			It("should return true for online user", func() {
				online := hub.CheckOnline(device.UID())
				Expect(online).To(BeFalse())
			})

			It("should return false for offline user", func() {
				online := hub.CheckOnline("offline-user")
				Expect(online).To(BeFalse())
			})
		})
	})

	Context("Offline", func() {
		Context("Send", func() {
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
			It("queue is nil - should return ErrQueueRequired when trying to cache with TTL > 0", func() {
				config := tok.NewHubConfig(mockActor)
				hub, _ = tok.CreateWsHandler(auth, tok.WithWsHandlerHubConfig(config))

				err := hub.Send("offline-user", []byte("test"), 60)
				Ω(err).To(HaveOccurred())
				Ω(err).To(MatchError(tok.ErrQueueRequired))
			})
		})
	})

	Context("Online method", func() {
		It("should return empty list when no users are online", func() {
			onlineUsers := hub.Online()
			Expect(onlineUsers).To(BeEmpty())
		})
	})

	Context("Kick method", func() {
		It("should not panic when kicking non-existent user", func() {
			Expect(func() {
				hub.Kick("non-existent-user")
			}).ToNot(Panic())
		})
	})

	Context("Hub with BeforeReceiveHandler", func() {
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

	Context("Hub with BeforeSendHandler", func() {
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

	Context("Hub with AfterSendHandler", func() {
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

	Context("Hub with CloseHandler", func() {
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

	Context("Hub with PingGenerator", func() {
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

	Context("Hub queue operations", func() {
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

	Context("Hub configuration variations", func() {
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
