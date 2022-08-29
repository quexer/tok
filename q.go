package tok

import (
	"context"
)

// Queue is FIFO queue interface, used by Hub
type Queue interface {
	Enq(ctx context.Context, uid interface{}, data []byte, ttl ...uint32) error
	Deq(ctx context.Context, uid interface{}) ([]byte, error)
	Len(ctx context.Context, uid interface{}) (int, error)
}
