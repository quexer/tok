package tok_test

import (
	"io"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"github.com/quexer/tok"
	"github.com/quexer/tok/mocks"
)

var _ = Describe("Custom Connection", func() {
	var (
		mockAdapter *mocks.MockConAdapter
		mockActor   *mocks.MockActor
		hub         *tok.Hub
		device      *tok.Device
	)

	BeforeEach(func() {
		mockAdapter = mocks.NewMockConAdapter(ctl)
		mockActor = mocks.NewMockActor(ctl)

		// Create hub with minimal config
		mockPing := mocks.NewMockPingGenerator(ctl)
		mockPing.EXPECT().Ping().Return([]byte("ping")).AnyTimes()

		// Add default mock queue to handle async operations
		mockQueue := mocks.NewMockQueue(ctl)
		mockQueue.EXPECT().Deq(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		mockQueue.EXPECT().Enq(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		config := tok.NewHubConfig(mockActor,
			tok.WithHubConfigPingProducer(mockPing),
			tok.WithHubConfigQueue(mockQueue))
		hub, _ = tok.CreateWsHandler(nil, tok.WithWsHandlerHubConfig(config))

		device = tok.CreateDevice("custom-user", "custom-session")
	})

	Describe("RegisterConnection", func() {
		It("should register custom connection successfully", func() {
			// Setup mock expectations
			// Read will be called in readLoop, simulate connection close after some time
			mockAdapter.EXPECT().Read().DoAndReturn(func() ([]byte, error) {
				time.Sleep(100 * time.Millisecond)
				return nil, io.EOF
			}).Times(1)
			mockAdapter.EXPECT().Close().Return(nil).AnyTimes()

			// Register the connection in a goroutine since readLoop blocks
			go hub.RegisterConnection(ctx, device, mockAdapter)

			// Give time for connection to be established
			time.Sleep(50 * time.Millisecond)

			// Verify device is online
			Expect(hub.CheckOnline(ctx, "custom-user")).To(BeTrue())
		})

		It("should receive messages through custom connection", func() {
			// Setup mock expectations
			msgData := []byte("hello from custom")

			// First Read returns our message, second Read blocks then returns EOF
			gomock.InOrder(
				mockAdapter.EXPECT().Read().Return(msgData, nil).Times(1),
				mockAdapter.EXPECT().Read().DoAndReturn(func() ([]byte, error) {
					time.Sleep(100 * time.Millisecond)
					return nil, io.EOF
				}).Times(1),
			)
			mockAdapter.EXPECT().Close().Return(nil).AnyTimes()

			// Expect actor to receive the message with any device matching our user
			mockActor.EXPECT().OnReceive(gomock.Any(), msgData).Do(func(dv *tok.Device, data []byte) {
				Expect(dv.UID()).To(Equal("custom-user"))
			}).Times(1)

			// Register the connection
			go hub.RegisterConnection(ctx, device, mockAdapter)

			// Give time for message to be processed
			time.Sleep(50 * time.Millisecond)
		})

		It("should send messages through custom connection", func() {
			// Setup mock expectations
			msgData := []byte("hello to custom")

			// Read blocks then returns EOF
			mockAdapter.EXPECT().Read().DoAndReturn(func() ([]byte, error) {
				time.Sleep(100 * time.Millisecond)
				return nil, io.EOF
			}).Times(1)
			mockAdapter.EXPECT().Close().Return(nil).AnyTimes()

			// Expect Write to be called with our message
			mockAdapter.EXPECT().Write(msgData).Return(nil).Times(1)

			// Register the connection
			go hub.RegisterConnection(ctx, device, mockAdapter)

			// Give time for connection to be established
			time.Sleep(50 * time.Millisecond)

			// Send a message
			err := hub.Send(ctx, "custom-user", msgData, 0)
			Expect(err).NotTo(HaveOccurred())

			// Give time for message to be sent
			time.Sleep(50 * time.Millisecond)
		})

		It("should handle connection close", func() {
			// Setup mock expectations
			// Read returns error immediately to simulate connection close
			mockAdapter.EXPECT().Read().Return(nil, io.EOF).Times(1)
			mockAdapter.EXPECT().Close().Return(nil).Times(1)

			// Register the connection
			go hub.RegisterConnection(ctx, device, mockAdapter)

			// Give time for connection to be established and then closed
			time.Sleep(100 * time.Millisecond)

			// Verify device is offline
			Expect(hub.CheckOnline(ctx, "custom-user")).To(BeFalse())
		})

		It("should handle ShareConn for SSO mode", func() {
			// Create another mock adapter
			mockAdapter2 := mocks.NewMockConAdapter(ctl)

			// Setup expectations for both adapters
			mockAdapter.EXPECT().Read().DoAndReturn(func() ([]byte, error) {
				time.Sleep(200 * time.Millisecond)
				return nil, io.EOF
			}).AnyTimes()
			mockAdapter.EXPECT().Close().Return(nil).AnyTimes()

			mockAdapter2.EXPECT().Read().DoAndReturn(func() ([]byte, error) {
				time.Sleep(200 * time.Millisecond)
				return nil, io.EOF
			}).AnyTimes()
			mockAdapter2.EXPECT().Close().Return(nil).AnyTimes()

			// Test ShareConn
			mockAdapter.EXPECT().ShareConn(mockAdapter2).Return(false).Times(1)
			mockAdapter.EXPECT().ShareConn(mockAdapter).Return(true).Times(1)

			// Register connections
			go hub.RegisterConnection(ctx, device, mockAdapter)
			time.Sleep(50 * time.Millisecond)

			// Verify ShareConn behavior
			Expect(mockAdapter.ShareConn(mockAdapter2)).To(BeFalse())
			Expect(mockAdapter.ShareConn(mockAdapter)).To(BeTrue())
		})

		It("should integrate with Queue for offline messages", func() {
			// Create mock queue
			mockQueue := mocks.NewMockQueue(ctl)

			// Create hub with queue
			mockPing := mocks.NewMockPingGenerator(ctl)
			mockPing.EXPECT().Ping().Return([]byte("ping")).AnyTimes()

			config := tok.NewHubConfig(mockActor,
				tok.WithHubConfigPingProducer(mockPing),
				tok.WithHubConfigQueue(mockQueue))
			hub, _ = tok.CreateWsHandler(nil, tok.WithWsHandlerHubConfig(config))

			// Setup expectations
			msgData := []byte("offline message")
			mockQueue.EXPECT().Deq(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
			mockQueue.EXPECT().Enq(gomock.Any(), "custom-user", msgData, uint32(300)).Return(nil).Times(1)

			// Send message with TTL while device is offline
			err := hub.Send(ctx, "custom-user", msgData, 300)
			Expect(err).NotTo(HaveOccurred())

			// Wait for async queue operation
			time.Sleep(50 * time.Millisecond)
		})
	})
})
