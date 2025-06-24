package tok_test

import (
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

})
