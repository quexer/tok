package tok_test

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/quexer/tok"
)

type closeAwareQueue struct {
	closed atomic.Bool
}

func (q *closeAwareQueue) Enq(_ context.Context, _ any, _ []byte, _ ...uint32) error {
	return nil
}

func (q *closeAwareQueue) Deq(_ context.Context, _ any) ([]byte, error) {
	return nil, nil
}

func (q *closeAwareQueue) Len(_ context.Context, _ any) (int, error) {
	return 0, nil
}

func (q *closeAwareQueue) Close() {
	q.closed.Store(true)
}

func (q *closeAwareQueue) IsClosed() bool {
	return q.closed.Load()
}

var _ = Describe("TCP Listen", func() {
	It("closes hub resources when listen fails", func() {
		q := &closeAwareQueue{}
		config, err := tok.NewHubConfig(noopActor{},
			tok.WithHubConfigQueue(q),
			tok.WithHubConfigReadTimeout(time.Second))
		Expect(err).To(Succeed())

		hub, err := tok.Listen(ctx, config, "invalid-addr", func([]byte) (*tok.Device, error) {
			return tok.CreateDevice("u", ""), nil
		})
		Expect(err).To(HaveOccurred())
		Expect(hub).To(BeNil())
		Eventually(q.IsClosed).To(BeTrue())
	})

	It("returns ErrAuthRequired when auth is nil", func() {
		q := &closeAwareQueue{}
		config, err := tok.NewHubConfig(noopActor{},
			tok.WithHubConfigQueue(q),
			tok.WithHubConfigReadTimeout(time.Second))
		Expect(err).To(Succeed())

		hub, err := tok.Listen(ctx, config, "127.0.0.1:0", nil)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, tok.ErrAuthRequired)).To(BeTrue())
		Expect(hub).To(BeNil())
		Expect(q.IsClosed()).To(BeFalse())
	})
})
