package tok

import (
	"context"
	"sync"
	"time"
)

type MemoryQueue struct {
	queues      sync.Map // uid -> *userQueue
	ctx         context.Context
	cancelFunc  context.CancelFunc
	IdleTimeout time.Duration // idle timeout before empty queue is removed, default 1 minute
}

type userQueue struct {
	mu         sync.Mutex
	items      []queueItem
	lastAccess time.Time // track last access time for cleanup
	deleted    bool      // set to true when removed from map by Cleanup
}

type queueItem struct {
	data       []byte
	expiration time.Time
}

func NewMemoryQueue() *MemoryQueue {
	ctx, cancel := context.WithCancel(context.Background())
	mq := &MemoryQueue{
		ctx:         ctx,
		cancelFunc:  cancel,
		IdleTimeout: time.Minute,
	}
	// Start cleanup routine
	go mq.cleanupRoutine()
	return mq
}

// Cleanup removes empty queues that have been idle longer than IdleTimeout.
func (mq *MemoryQueue) Cleanup() {
	now := time.Now()
	mq.queues.Range(func(key, value interface{}) bool {
		queue := value.(*userQueue)
		queue.mu.Lock()
		if len(queue.items) == 0 && now.Sub(queue.lastAccess) > mq.IdleTimeout {
			queue.deleted = true
			mq.queues.Delete(key)
		}
		queue.mu.Unlock()
		return true
	})
}

// cleanupRoutine periodically cleans up empty queues
func (mq *MemoryQueue) cleanupRoutine() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-mq.ctx.Done():
			return
		case <-ticker.C:
			mq.Cleanup()
		}
	}
}

// Close stops the cleanup routine
func (mq *MemoryQueue) Close() {
	if mq.cancelFunc != nil {
		mq.cancelFunc()
	}
}

func (mq *MemoryQueue) Enq(ctx context.Context, uid interface{}, data []byte, ttl ...uint32) error {
	for {
		qu, _ := mq.queues.LoadOrStore(uid, &userQueue{lastAccess: time.Now()})
		queue := qu.(*userQueue)

		queue.mu.Lock()

		// If Cleanup has marked this queue as deleted, the queue object is detached
		// from the map. Discard it and retry with a fresh entry.
		if queue.deleted {
			queue.mu.Unlock()
			mq.queues.CompareAndDelete(uid, qu)
			continue
		}

		queue.lastAccess = time.Now()

		var expiration time.Time
		if len(ttl) > 0 && ttl[0] > 0 {
			expiration = time.Now().Add(time.Duration(ttl[0]) * time.Second)
		}

		queue.items = append(queue.items, queueItem{
			data:       data,
			expiration: expiration,
		})
		queue.mu.Unlock()
		return nil
	}
}

func (mq *MemoryQueue) Deq(ctx context.Context, uid interface{}) ([]byte, error) {
	qu, ok := mq.queues.Load(uid)
	if !ok {
		return nil, nil
	}

	queue := qu.(*userQueue)
	queue.mu.Lock()
	defer queue.mu.Unlock()

	queue.lastAccess = time.Now()

	// Clean up expired items
	mq.clearExpireItem(queue)

	if len(queue.items) == 0 {
		// Don't delete immediately, let cleanup routine handle it
		return nil, nil
	}

	// Get the first valid element
	data := queue.items[0].data
	queue.items = queue.items[1:]

	return data, nil
}

func (mq *MemoryQueue) clearExpireItem(queue *userQueue) {
	// Clean up all expired items
	now := time.Now()
	validItems := queue.items[:0]
	for _, item := range queue.items {
		if item.expiration.IsZero() || item.expiration.After(now) {
			validItems = append(validItems, item)
		}
	}
	queue.items = validItems
}

func (mq *MemoryQueue) Len(ctx context.Context, uid interface{}) (int, error) {
	qu, ok := mq.queues.Load(uid)
	if !ok {
		return 0, nil
	}

	queue := qu.(*userQueue)
	queue.mu.Lock()
	defer queue.mu.Unlock()

	queue.lastAccess = time.Now()

	// Clean up expired items
	mq.clearExpireItem(queue)

	// Don't delete empty queue immediately, let cleanup routine handle it

	return len(queue.items), nil
}
