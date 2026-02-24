package tok_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/quexer/tok"
)

var _ = Describe("MemoryQ", func() {
	Ω(1).To(Equal(1))

	var queue *tok.MemoryQueue
	BeforeEach(func() {
		queue = tok.NewMemoryQueue()

		f := func(uid, data string, ttl ...uint32) {
			// 入队
			err := queue.Enq(ctx, uid, []byte(data), ttl...)
			Ω(err).To(Succeed())
		}

		f("u1", "d1")
		f("u1", "d11")
		f("u2", "d2")

	})

	It("Enq", func() {
		err := queue.Enq(ctx, "u1", []byte("d12"), 1)
		Ω(err).To(Succeed())
		count, err := queue.Len(ctx, "u1")
		Ω(err).To(Succeed())
		Ω(count).To(Equal(3))

		time.Sleep(2 * time.Second)

		count, err = queue.Len(ctx, "u1")
		Ω(err).To(Succeed())
		Ω(count).To(Equal(2))

	})

	It("Deq", func() {
		data, err := queue.Deq(ctx, "u1")
		Ω(err).To(Succeed())
		Ω(data).To(Equal([]byte("d1")))
	})

	It("Len", func() {
		count, err := queue.Len(ctx, "u1")
		Ω(err).To(Succeed())
		Ω(count).To(Equal(2))
	})

	It("should not lose data during concurrent Enq and Cleanup", func() {
		mq := tok.NewMemoryQueue()
		defer mq.Close()
		// Use zero idle timeout so Cleanup always removes empty queues
		mq.IdleTimeout = 0

		const uid = "race-user"
		var wg sync.WaitGroup
		var lostCount int64
		done := make(chan struct{})

		// goroutine A: continuously call Cleanup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					mq.Cleanup()
				}
			}
		}()

		// goroutine B: Enq + verify Len > 0
		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			defer wg.Done()
			for i := range 10000 {
				// Make queue empty and eligible for cleanup
				_ = mq.Enq(ctx, uid, []byte("setup"))
				_, _ = mq.Deq(ctx, uid)

				// Enq new data — races with Cleanup deleting the key
				_ = mq.Enq(ctx, uid, fmt.Appendf(nil, "important-%d", i))

				// Verify: Len must be > 0 after a successful Enq
				n, _ := mq.Len(ctx, uid)
				if n == 0 {
					atomic.AddInt64(&lostCount, 1)
				}

				// drain for next iteration
				_, _ = mq.Deq(ctx, uid)
			}
			close(done)
		}()

		wg.Wait()
		Expect(atomic.LoadInt64(&lostCount)).To(BeZero(),
			"data lost due to cleanup race condition")
	})

})
