package tok

import (
	"context"
	"errors"
	"sync"
	"time"
)

type MemoryQueue struct {
	queues sync.Map // uid -> *userQueue
}

type userQueue struct {
	mu    sync.Mutex
	items []queueItem
}

type queueItem struct {
	data       []byte
	expiration time.Time
}

func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{}
}

func (mq *MemoryQueue) Enq(ctx context.Context, uid interface{}, data []byte, ttl ...uint32) error {
	qu, _ := mq.queues.LoadOrStore(uid, &userQueue{})
	queue := qu.(*userQueue)

	queue.mu.Lock()
	defer queue.mu.Unlock()

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
		return nil, errors.New("queue not found")
	}

	queue := qu.(*userQueue)
	queue.mu.Lock()
	defer queue.mu.Unlock()

	// 清理过期元素
	mq.clearExpireItem(queue)

	if len(queue.items) == 0 {
		mq.queues.Delete(uid)
		return nil, errors.New("queue is empty")
	}

	// 取出第一个有效元素
	data := queue.items[0].data
	if len(queue.items) == 1 {
		mq.queues.Delete(uid)
	} else {
		queue.items = queue.items[1:]
	}

	return data, nil
}

func (mq *MemoryQueue) clearExpireItem(queue *userQueue) {
	// 清理所有过期元素
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

	// 清理过期元素
	mq.clearExpireItem(queue)

	// 自动删除空队列
	if len(queue.items) == 0 {
		mq.queues.Delete(uid)
	}

	return len(queue.items), nil
}
