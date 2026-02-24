package tok

import (
	"context"
)

//go:generate mockgen -destination=mocks/q.go -package=mocks . Queue

// Queue is FIFO queue interface, used by Hub
type Queue interface {
	Enq(ctx context.Context, uid any, data []byte, ttl ...uint32) error
	Deq(ctx context.Context, uid any) ([]byte, error)
	Len(ctx context.Context, uid any) (int, error)
}
