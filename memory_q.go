package tok

import (
	"context"
	"sync"
	"time"
)

type MemoryQueue struct {
	queues     sync.Map // uid -> *userQueue
	ctx        context.Context
	cancelFunc context.CancelFunc
}

type userQueue struct {
	mu         sync.Mutex
	items      []queueItem
	lastAccess time.Time // track last access time for cleanup
}

type queueItem struct {
	data       []byte
	expiration time.Time
}

func NewMemoryQueue() *MemoryQueue {
	ctx, cancel := context.WithCancel(context.Background())
	mq := &MemoryQueue{
		ctx:        ctx,
		cancelFunc: cancel,
	}
	// Start cleanup routine
	go mq.cleanupRoutine()
	return mq
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
			now := time.Now()
			mq.queues.Range(func(key, value interface{}) bool {
				queue := value.(*userQueue)
				queue.mu.Lock()
				// Remove queue if empty and not accessed for 5 minutes
				if len(queue.items) == 0 && now.Sub(queue.lastAccess) > time.Minute {
					queue.mu.Unlock()
					mq.queues.Delete(key)
				} else {
					queue.mu.Unlock()
				}
				return true
			})
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
	qu, _ := mq.queues.LoadOrStore(uid, &userQueue{lastAccess: time.Now()})
	queue := qu.(*userQueue)

	queue.mu.Lock()
	defer queue.mu.Unlock()

	queue.lastAccess = time.Now()

	var expiration time.Time
	if len(ttl) > 0 && ttl[0] > 0 {
		expiration = time.Now().Add(time.Duration(ttl[0]) * time.Second)
	}

	queue.items = append(queue.items, queueItem{
		data:       data,
		expiration: expiration,
	})
	return nil
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
