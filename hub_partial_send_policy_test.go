package tok_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/quexer/tok"
)

type noopActor struct{}

func (noopActor) OnReceive(_ *tok.Device, _ []byte) {}

type recordingQueue struct {
	mu   sync.Mutex
	enqs int
}

func (q *recordingQueue) Enq(_ context.Context, _ interface{}, _ []byte, _ ...uint32) error {
	q.mu.Lock()
	q.enqs++
	q.mu.Unlock()
	return nil
}

func (q *recordingQueue) Deq(_ context.Context, _ interface{}) ([]byte, error) {
	return nil, nil
}

func (q *recordingQueue) Len(_ context.Context, _ interface{}) (int, error) {
	return 0, nil
}

func (q *recordingQueue) EnqCount() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.enqs
}

type blockingAdapter struct {
	readStarted chan struct{}
	readBlock   chan struct{}
	writeErr    error
	writeCount  atomic.Int32
}

func newBlockingAdapter(writeErr error) *blockingAdapter {
	return &blockingAdapter{
		readStarted: make(chan struct{}),
		readBlock:   make(chan struct{}),
		writeErr:    writeErr,
	}
}

func (a *blockingAdapter) Read() ([]byte, error) {
	select {
	case <-a.readStarted:
	default:
		close(a.readStarted)
	}
	<-a.readBlock
	return nil, io.EOF
}

func (a *blockingAdapter) Write(_ []byte) error {
	a.writeCount.Add(1)
	return a.writeErr
}

func (a *blockingAdapter) Close() error {
	select {
	case <-a.readBlock:
	default:
		close(a.readBlock)
	}
	return nil
}

func (a *blockingAdapter) ShareConn(adapter tok.ConAdapter) bool {
	other, ok := adapter.(*blockingAdapter)
	if !ok {
		return false
	}
	return a == other
}

func (a *blockingAdapter) WriteCount() int {
	return int(a.writeCount.Load())
}

var _ = Describe("Hub partial send policy", func() {
	const uid = "partial-user"
	const ttl = uint32(30)

	var (
		hub   *tok.Hub
		queue *recordingQueue
	)

	createHub := func(extraOpts ...tok.HubConfigOption) {
		opts := []tok.HubConfigOption{
			tok.WithHubConfigQueue(queue),
			tok.WithHubConfigSso(false),
			tok.WithHubConfigReadTimeout(time.Second),
		}
		opts = append(opts, extraOpts...)

		config, err := tok.NewHubConfig(noopActor{}, opts...)
		Expect(err).To(Succeed())

		hub, _, err = tok.CreateWsHandler(ctx, func(_ *http.Request) (*tok.Device, error) {
			return tok.CreateDevice("setup-user", "setup-device"), nil
		}, tok.WithWsHandlerHubConfig(config))
		Expect(err).To(Succeed())
	}

	registerPair := func(firstErr, secondErr error) (*blockingAdapter, *blockingAdapter) {
		a1 := newBlockingAdapter(firstErr)
		a2 := newBlockingAdapter(secondErr)

		go hub.RegisterConnection(ctx, tok.CreateDevice(uid, "dv-1"), a1)
		go hub.RegisterConnection(ctx, tok.CreateDevice(uid, "dv-2"), a2)

		Eventually(a1.readStarted).Should(BeClosed())
		Eventually(a2.readStarted).Should(BeClosed())
		Eventually(func() bool {
			return hub.CheckOnline(ctx, uid)
		}).Should(BeTrue())

		return a1, a2
	}

	BeforeEach(func() {
		queue = &recordingQueue{}
	})

	AfterEach(func() {
		if hub != nil {
			hub.Close()
		}
	})

	It("defaults to cache on any failure for partial delivery", func() {
		createHub()
		failErr := errors.New("write failed")
		a1, a2 := registerPair(failErr, nil)

		err := hub.Send(ctx, uid, []byte("payload"), ttl)
		Expect(err).To(Succeed())
		Expect(queue.EnqCount()).To(Equal(1))
		Expect(a1.WriteCount()).To(Equal(1))
		Expect(a2.WriteCount()).To(Equal(1))
	})

	It("does not cache partial delivery when policy is cache-when-all-failed", func() {
		createHub(tok.WithHubConfigPartialSendPolicy(tok.PartialSendCacheWhenAllFailed))
		failErr := errors.New("write failed")
		a1, a2 := registerPair(failErr, nil)

		err := hub.Send(ctx, uid, []byte("payload"), ttl)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, tok.ErrPartialDelivered)).To(BeTrue())
		Expect(queue.EnqCount()).To(Equal(0))
		Expect(a1.WriteCount()).To(Equal(1))
		Expect(a2.WriteCount()).To(Equal(1))
	})

	It("still caches when all online sends fail under cache-when-all-failed policy", func() {
		createHub(tok.WithHubConfigPartialSendPolicy(tok.PartialSendCacheWhenAllFailed))
		failErr := errors.New("write failed")
		registerPair(failErr, failErr)

		err := hub.Send(ctx, uid, []byte("payload"), ttl)
		Expect(err).To(Succeed())
		Expect(queue.EnqCount()).To(Equal(1))
	})

	It("never caches online send failures under no-cache policy", func() {
		createHub(tok.WithHubConfigPartialSendPolicy(tok.PartialSendNoCache))
		failErr := errors.New("write failed")
		registerPair(failErr, failErr)

		err := hub.Send(ctx, uid, []byte("payload"), ttl)
		Expect(err).To(HaveOccurred())
		Expect(queue.EnqCount()).To(Equal(0))
	})
})
