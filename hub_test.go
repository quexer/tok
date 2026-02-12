package tok_test

import (
	"context"
	"fmt"
	"io"
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

		hub *tok.Hub

		server    *httptest.Server
		wsURL     string
		dialer    *websocket.Dialer
		hubConfig *tok.HubConfig
	)

	BeforeEach(func() {
		mockActor = mocks.NewMockActor(ctl)
		mockQueue = mocks.NewMockQueue(ctl)
		mockPingGen = mocks.NewMockPingGenerator(ctl)

		var err error
		hubConfig, err = tok.NewHubConfig(mockActor,
			tok.WithHubConfigQueue(mockQueue),
			tok.WithHubConfigPingProducer(mockPingGen), // Required to avoid fatal error
		)
		Ω(err).To(Succeed())

		dialer = &websocket.Dialer{}
	})

	// JustBeforeEach creates the hub and server using the configuration from BeforeEach.
	// This allows nested BeforeEach blocks to modify the config before the hub is created.
	const uid = "test-user"
	JustBeforeEach(func() {
		var handler http.Handler
		// Create default auth and hub config
		auth := func(_ *http.Request) (*tok.Device, error) {
			return tok.CreateDevice(uid, "dv-id"), nil
		}

		var err error
		hub, handler, err = tok.CreateWsHandler(ctx, auth,
			tok.WithWsHandlerHubConfig(hubConfig),
			tok.WithWsHandlerEngine(tok.WsEngineGorilla)) // Use Gorilla engine to match client
		Ω(err).To(Succeed())

		// Setup test server
		server = httptest.NewServer(handler)
		wsURL = "ws" + server.URL[4:] // Convert http:// to ws://
	})

	AfterEach(func() {
		hub.Close()
		server.Close()
	})

	Describe("Send", func() {
		It("should send message to online device", func() {
			// deq once when go online
			mockQueue.EXPECT().Deq(gomock.Any(), gomock.Any())
			// Connect websocket client
			ws, _, err := dialer.Dial(wsURL, nil)
			Expect(err).NotTo(HaveOccurred())
			defer ws.Close()

			// Give connection time to establish
			time.Sleep(50 * time.Millisecond)

			// Send message through hub
			err = hub.Send(ctx, uid, []byte("test message"), 0)
			Expect(err).NotTo(HaveOccurred())

			// Read message from websocket
			_, msg, err := ws.ReadMessage()
			Expect(err).NotTo(HaveOccurred())
			Expect(msg).To(Equal([]byte("test message")))
		})

		It("should return error when device is offline and no queue", func() {
			// No websocket connection established
			err := hub.Send(ctx, "offline-user", []byte("test message"), 0)
			Expect(err).To(Equal(tok.ErrOffline))
		})

		It("should queue message when device is offline with TTL", func() {
			// Setup mock expectation for queue - use AnyTimes() for async call
			mockQueue.EXPECT().Enq(gomock.Any(), "offline-user", []byte("queued message"), gomock.Any())

			// Send with TTL > 0 to trigger queueing
			err := hub.Send(ctx, "offline-user", []byte("queued message"), 300)
			Expect(err).NotTo(HaveOccurred())

		})

		It("should handle queue error gracefully", func() {
			// Setup mock to return error
			mockQueue.EXPECT().Enq(gomock.Any(), "offline-user", []byte("failed message"), gomock.Any()).Return(context.DeadlineExceeded).AnyTimes()

			// Send with TTL > 0
			err := hub.Send(ctx, "offline-user", []byte("failed message"), 300)
			Expect(err).To(HaveOccurred()) // Send returns nil even if queue fails
		})
	})

	Describe("CheckOnline", func() {
		It("should return false when device is offline", func() {
			online := hub.CheckOnline(ctx, "offline-user")
			Expect(online).To(BeFalse())
		})

		It("should return true when device is online", func() {
			mockQueue.EXPECT().Deq(gomock.Any(), gomock.Any())
			// Connect websocket client, will be authenticated as "test-user"
			ws, _, err := dialer.Dial(wsURL, nil)
			Expect(err).NotTo(HaveOccurred())
			defer ws.Close()

			// Give connection time to establish
			time.Sleep(50 * time.Millisecond)

			online := hub.CheckOnline(ctx, uid)
			Expect(online).To(BeTrue())
		})
	})

	Describe("Online", func() {
		It("should return empty list when no devices online", func() {
			devices := hub.Online(ctx)
			Expect(devices).To(BeEmpty())
		})

		It("should return list of online devices", func() {
			mockQueue.EXPECT().Deq(gomock.Any(), gomock.Any())
			// Since auth function always returns "test-user", let's just test single connection
			ws, _, err := dialer.Dial(wsURL, nil)
			Expect(err).NotTo(HaveOccurred())
			defer ws.Close()

			// Give connection time to establish
			time.Sleep(50 * time.Millisecond)

			userList := hub.Online(ctx)
			Expect(userList).To(Equal([]any{uid}))
		})
	})

	Describe("Kick", func() {
		BeforeEach(func() {
			// Create hub with bye generator
			mockByeGen := mocks.NewMockByeGenerator(ctl)
			mockByeGen.EXPECT().Bye(gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("bye")).AnyTimes()

			var err error
			hubConfig, err = tok.NewHubConfig(mockActor,
				tok.WithHubConfigQueue(mockQueue),
				tok.WithHubConfigPingProducer(mockPingGen),
				tok.WithHubConfigByeGenerator(mockByeGen),
			)
			Ω(err).To(Succeed())
		})

		It("should disconnect device when kicked", func() {
			mockQueue.EXPECT().Deq(gomock.Any(), gomock.Any())
			// Connect websocket client
			ws, _, err := dialer.Dial(wsURL, nil)
			Expect(err).NotTo(HaveOccurred())
			defer ws.Close()

			// Give connection time to establish
			time.Sleep(50 * time.Millisecond)

			// Verify device is online
			Expect(hub.CheckOnline(ctx, uid)).To(BeTrue())

			// Kick the device
			hub.Kick(ctx, uid)

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
			Expect(hub.CheckOnline(ctx, uid)).To(BeFalse())
		})

		It("should handle kicking offline device gracefully", func() {
			// No error should occur when kicking offline device
			hub.Kick(ctx, "offline-user")
		})
	})

	Describe("Hub with SSO", func() {
		BeforeEach(func() {
			var err error
			hubConfig, err = tok.NewHubConfig(mockActor,
				tok.WithHubConfigQueue(mockQueue),
				tok.WithHubConfigPingProducer(mockPingGen),
				tok.WithHubConfigSso(true), // Enable SSO
			)
			Ω(err).To(Succeed())
		})

		It("should disconnect old connection when new one arrives", func() {
			mockQueue.EXPECT().Deq(gomock.Any(), gomock.Any()).Times(2)
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
			err = hub.Send(ctx, uid, []byte("test"), 0)
			Expect(err).NotTo(HaveOccurred())

			_, msg, err := ws2.ReadMessage()
			Expect(err).NotTo(HaveOccurred())
			Expect(msg).To(Equal([]byte("test")))
		})
	})

	newBlockingAdapter := func() *mocks.MockConAdapter {
		a := mocks.NewMockConAdapter(ctl)
		a.EXPECT().Read().DoAndReturn(func() ([]byte, error) {
			time.Sleep(500 * time.Millisecond)
			return nil, io.EOF
		}).AnyTimes()
		a.EXPECT().Close().Return(nil).AnyTimes()
		a.EXPECT().ShareConn(gomock.Any()).Return(false).AnyTimes()
		return a
	}

	Describe("Close", func() {
		It("should return ErrHubClosed from Send after Close", func() {
			hub.Close()

			err := hub.Send(ctx, uid, []byte("test"), 0)
			Expect(err).To(Equal(tok.ErrHubClosed))
		})

		It("should return false from CheckOnline after Close", func() {
			mockQueue.EXPECT().Deq(gomock.Any(), gomock.Any())
			// Connect websocket client
			ws, _, err := dialer.Dial(wsURL, nil)
			Expect(err).NotTo(HaveOccurred())
			defer ws.Close()

			// Give connection time to establish
			time.Sleep(50 * time.Millisecond)

			// Verify device is online before close
			Expect(hub.CheckOnline(ctx, uid)).To(BeTrue())

			hub.Close()

			// After close, CheckOnline should return false
			Expect(hub.CheckOnline(ctx, uid)).To(BeFalse())
		})

		It("should return empty from Online after Close", func() {
			mockQueue.EXPECT().Deq(gomock.Any(), gomock.Any())
			// Connect websocket client
			ws, _, err := dialer.Dial(wsURL, nil)
			Expect(err).NotTo(HaveOccurred())
			defer ws.Close()

			// Give connection time to establish
			time.Sleep(50 * time.Millisecond)

			// Verify there are online users before close
			Expect(hub.Online(ctx)).NotTo(BeEmpty())

			hub.Close()

			// After close, Online should return nil
			Expect(hub.Online(ctx)).To(BeNil())
		})

		It("should disconnect existing connections on Close", func() {
			mockQueue.EXPECT().Deq(gomock.Any(), gomock.Any())
			// Connect websocket client
			ws, _, err := dialer.Dial(wsURL, nil)
			Expect(err).NotTo(HaveOccurred())
			defer ws.Close()

			// Give connection time to establish
			time.Sleep(50 * time.Millisecond)

			hub.Close()

			// WebSocket client should get read error since connection was closed
			_, _, err = ws.ReadMessage()
			Expect(err).To(HaveOccurred())
		})

		It("should be safe to call Close multiple times", func() {
			// Close multiple times should not panic
			hub.Close()
			hub.Close()
			hub.Close()
		})
	})

	Describe("byeThenClose", func() {
		BeforeEach(func() {
			mockQueue.EXPECT().Deq(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
			mockPingGen.EXPECT().Ping().Return([]byte("ping")).AnyTimes()

			mockBeforeSend := mocks.NewMockBeforeSendHandler(ctl)
			mockBeforeSend.EXPECT().BeforeSend(gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("encode error")).AnyTimes()

			mockByeGen := mocks.NewMockByeGenerator(ctl)
			mockByeGen.EXPECT().Bye(gomock.Any(), gomock.Any(), gomock.Any()).
				Return([]byte("bye")).AnyTimes()

			var err error
			hubConfig, err = tok.NewHubConfig(mockActor,
				tok.WithHubConfigQueue(mockQueue),
				tok.WithHubConfigPingProducer(mockPingGen),
				tok.WithHubConfigSso(true),
				tok.WithHubConfigBeforeSend(mockBeforeSend),
				tok.WithHubConfigByeGenerator(mockByeGen),
			)
			Ω(err).To(Succeed())
		})

		It("should not write when beforeSend returns error", func() {
			// No Write expectation on adapter1.
			// If Write is called, gomock will report "unexpected call" and fail.
			adapter1 := newBlockingAdapter()
			adapter2 := newBlockingAdapter()

			go hub.RegisterConnection(ctx, tok.CreateDevice("sso-user", "device-1"), adapter1)
			time.Sleep(50 * time.Millisecond)

			// Same UID triggers byeThenClose on first connection
			go hub.RegisterConnection(ctx, tok.CreateDevice("sso-user", "device-2"), adapter2)
			time.Sleep(100 * time.Millisecond)
		})
	})
})
