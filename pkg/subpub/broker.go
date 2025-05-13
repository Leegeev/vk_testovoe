package subpub

import (
	"context"
	"sync"
)

// broker реализует SubPub и управляет набором тем (topics).
type broker struct {
	mu     sync.RWMutex      // защищает topics и флаг closed
	topics map[string]*topic // темы по имени subject
	closed bool              // true после вызова Close
	wg     sync.WaitGroup    // ждёт dispatcher-горутины
}

// NewSubPub создаёт и возвращает новую шину событий.
func NewSubPub() SubPub {
	return &broker{
		topics: make(map[string]*topic),
	}
}

// Subscribe регистрирует подписчика на subject.
func (b *broker) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, ErrClosed
	}
	b.mu.RUnlock()
	if subject == "" {
		return nil, ErrTopicNameIsEmpty
	}
	t := b.getOrCreateTopic(subject)
	sub := newSubscription(cb)
	t.addSubscriber(sub)
	return sub, nil
}

// Publish публикует сообщение msg в тему subject.
// Если шина закрыта — ErrClosed. Если темы нет — ErrTopictNotFound
func (b *broker) Publish(subject string, msg interface{}) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return ErrClosed
	}
	t, ok := b.topics[subject]
	b.mu.RUnlock()
	if !ok {
		return ErrTopictNotFound
	}
	t.publish(msg)
	return nil
}

// Close корректно завершает работу шины или выходит по ctx.
func (b *broker) Close(ctx context.Context) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return ErrClosed
	}
	b.closed = true
	// закрываем все темы
	for _, t := range b.topics {
		t.close()
	}
	b.mu.Unlock()

	// ждём завершения dispatcher-горутин
	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// getOrCreateTopic возвращает тему или создаёт новую.
func (b *broker) getOrCreateTopic(subject string) *topic {
	b.mu.Lock()
	defer b.mu.Unlock()

	t, ok := b.topics[subject]
	if !ok {
		t = newTopic(subject, &b.wg)
		b.topics[subject] = t
	}
	return t
}
