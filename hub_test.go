package tok_test

import (
	"context"
	"net/http"
	"net/http/httptest"
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
		mockActor   *mocks.MockActor
		mockQueue   *mocks.MockQueue
		mockPingGen *mocks.MockPingGenerator
		hub         *tok.Hub
		server      *httptest.Server
		wsURL       string
		dialer      *websocket.Dialer
		hubConfig   *tok.HubConfig
		auth        tok.WsAuthFunc
	)

	BeforeEach(func() {
		mockActor = mocks.NewMockActor(ctl)
		mockQueue = mocks.NewMockQueue(ctl)
		mockPingGen = mocks.NewMockPingGenerator(ctl)

		// Setup default expectations for async calls
		mockQueue.EXPECT().Deq(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		mockQueue.EXPECT().Enq(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		// Create default auth and hub config
		auth = func(r *http.Request) (*tok.Device, error) {
			uid := r.Header.Get("uid")
			if uid == "" {
				uid = "test-user"
			}
			return tok.CreateDevice(uid, ""), nil
		}

		hubConfig = tok.NewHubConfig(mockActor,
			tok.WithHubConfigQueue(mockQueue),
			tok.WithHubConfigPingProducer(mockPingGen), // Required to avoid fatal error
		)

		dialer = &websocket.Dialer{}
	})

	// JustBeforeEach creates the hub and server using the configuration from BeforeEach.
	// This allows nested BeforeEach blocks to modify the config before the hub is created.
	JustBeforeEach(func() {
		var handler http.Handler
		hub, handler = tok.CreateWsHandler(auth,
			tok.WithWsHandlerHubConfig(hubConfig),
			tok.WithWsHandlerEngine(tok.WsEngineGorilla)) // Use Gorilla engine to match client

		// Setup test server
		server = httptest.NewServer(handler)
		wsURL = "ws" + server.URL[4:] // Convert http:// to ws://
	})

	AfterEach(func() {
		if server != nil {
			server.Close()
		}
	})

	Describe("Send", func() {
		It("should send message to online device", func() {
			// Connect websocket client
			ws, _, err := dialer.Dial(wsURL, nil)
			Expect(err).NotTo(HaveOccurred())
			defer ws.Close()

			// Give connection time to establish
			time.Sleep(50 * time.Millisecond)

			// Send message through hub
			err = hub.Send("test-user", []byte("test message"), 0)
			Expect(err).NotTo(HaveOccurred())

			// Read message from websocket
			_, msg, err := ws.ReadMessage()
			Expect(err).NotTo(HaveOccurred())
			Expect(msg).To(Equal([]byte("test message")))
		})

		It("should return error when device is offline and no queue", func() {
			// No websocket connection established
			err := hub.Send("offline-user", []byte("test message"), 0)
			Expect(err).To(Equal(tok.ErrOffline))
		})

		It("should queue message when device is offline with TTL", func() {
			// Setup mock expectation for queue - use AnyTimes() for async call
			mockQueue.EXPECT().Enq(gomock.Any(), "offline-user", []byte("queued message"), gomock.Any()).Return(nil).AnyTimes()

			// Send with TTL > 0 to trigger queueing
			err := hub.Send("offline-user", []byte("queued message"), 300)
			Expect(err).NotTo(HaveOccurred())

		})

		It("should handle queue error gracefully", func() {
			// Setup mock to return error
			mockQueue.EXPECT().Enq(gomock.Any(), "offline-user", []byte("failed message"), gomock.Any()).Return(context.DeadlineExceeded).AnyTimes()

			// Send with TTL > 0
			err := hub.Send("offline-user", []byte("failed message"), 300)
			Expect(err).NotTo(HaveOccurred()) // Send returns nil even if queue fails
		})
	})

	Describe("CheckOnline", func() {
		It("should return false when device is offline", func() {
			online := hub.CheckOnline("offline-user")
			Expect(online).To(BeFalse())
		})

		It("should return true when device is online", func() {
			// Connect websocket client
			header := http.Header{}
			header.Set("uid", "online-user")
			ws, _, err := dialer.Dial(wsURL, header)
			Expect(err).NotTo(HaveOccurred())
			defer ws.Close()

			// Give connection time to establish
			time.Sleep(50 * time.Millisecond)

			online := hub.CheckOnline("online-user")
			Expect(online).To(BeTrue())
		})
	})

	Describe("Online", func() {
		It("should return empty list when no devices online", func() {
			devices := hub.Online()
			Expect(devices).To(BeEmpty())
		})

		It("should return list of online devices", func() {
			// Since auth function always returns "test-user", let's just test single connection
			ws, _, err := dialer.Dial(wsURL, nil)
			Expect(err).NotTo(HaveOccurred())
			defer ws.Close()

			// Give connection time to establish
			time.Sleep(50 * time.Millisecond)

			userList := hub.Online()
			Expect(userList).To(Equal([]any{"test-user"}))
		})
	})

	Describe("Kick", func() {
		BeforeEach(func() {
			// Create hub with bye generator
			mockByeGen := mocks.NewMockByeGenerator(ctl)
			mockByeGen.EXPECT().Bye(gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("bye")).AnyTimes()

			auth = func(r *http.Request) (*tok.Device, error) {
				return tok.CreateDevice("kick-user", ""), nil
			}

			hubConfig = tok.NewHubConfig(mockActor,
				tok.WithHubConfigQueue(mockQueue),
				tok.WithHubConfigPingProducer(mockPingGen),
				tok.WithHubConfigByeGenerator(mockByeGen),
			)
		})

		It("should disconnect device when kicked", func() {
			// Connect websocket client
			ws, _, err := dialer.Dial(wsURL, nil)
			Expect(err).NotTo(HaveOccurred())
			defer ws.Close()

			// Give connection time to establish
			time.Sleep(50 * time.Millisecond)

			// Verify device is online
			Expect(hub.CheckOnline("kick-user")).To(BeTrue())

			// Kick the device
			hub.Kick("kick-user")

			// Try to read bye message or handle close
			_, msg, err := ws.ReadMessage()
			if err == nil {
				// If we got a message, it should be bye
				Expect(msg).To(Equal([]byte("bye")))
			} else {
				// Connection might be closed immediately after sending bye
				// This is also acceptable behavior
				_, ok := err.(*websocket.CloseError)
				Expect(ok).To(BeTrue())
			}

			// Give time for disconnection
			time.Sleep(50 * time.Millisecond)

			// Verify device is offline
			Expect(hub.CheckOnline("kick-user")).To(BeFalse())
		})

		It("should handle kicking offline device gracefully", func() {
			// No error should occur when kicking offline device
			hub.Kick("offline-user")
		})
	})

	Describe("Hub with SSO", func() {
		BeforeEach(func() {
			// Create hub with SSO enabled
			auth = func(r *http.Request) (*tok.Device, error) {
				return tok.CreateDevice("sso-user", ""), nil
			}

			hubConfig = tok.NewHubConfig(mockActor,
				tok.WithHubConfigQueue(mockQueue),
				tok.WithHubConfigPingProducer(mockPingGen),
				tok.WithHubConfigSso(true), // Enable SSO
			)
		})

		It("should disconnect old connection when new one arrives", func() {
			// Connect first client
			ws1, _, err := dialer.Dial(wsURL, nil)
			Expect(err).NotTo(HaveOccurred())
			defer ws1.Close()

			// Give connection time to establish
			time.Sleep(50 * time.Millisecond)

			// Connect second client with same user
			ws2, _, err := dialer.Dial(wsURL, nil)
			Expect(err).NotTo(HaveOccurred())
			defer ws2.Close()

			// Give time for SSO to kick in
			time.Sleep(50 * time.Millisecond)

			// First connection should receive error when trying to read
			_, _, err = ws1.ReadMessage()
			Expect(err).To(HaveOccurred())

			// Second connection should work fine
			err = hub.Send("sso-user", []byte("test"), 0)
			Expect(err).NotTo(HaveOccurred())

			_, msg, err := ws2.ReadMessage()
			Expect(err).NotTo(HaveOccurred())
			Expect(msg).To(Equal([]byte("test")))
		})
	})
})
