package subpub

import (
	"context"
	"sync"
)

// broker реализует интерфейс SubPub.
type broker struct {
	mu     sync.RWMutex      // защищает map topics и флаг closed
	topics map[string]*topic // одна тема = одна очередь FIFO
	closed bool              // true после Close
	wg     sync.WaitGroup    // ждём ВСЕ внутренние goroutine
}

func NewSubPub() SubPub {
	panic("Implement me")
}

func (b *broker) Subscribe(subject string, cb MessageHandler) (Subscription, error) {

	b.mu.RLock()
	defer b.mu.RUnlock()

	// если брокер уже закрыт
	if b.closed {
		return nil, ErrClosed
	}

	// если брокер открыт
	t := b.getOrCreateTopic(subject)
	sub := newSubscription(cb)
	t.addSubscriber(sub)
	return sub, nil
}

func (b *broker) Publish(subject string, msg interface{}) error {
	return nil
}

func (b *broker) Close(ctx context.Context) error {
	return nil
}

func (b *broker) getOrCreateTopic(name string) topic {
	return nil
}
