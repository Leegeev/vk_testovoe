// file: broker.go
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

	t := b.getOrCreateTopic(subject)
	sub := newSubscription(cb)
	t.addSubscriber(sub)
	return sub, nil
}

// Publish публикует сообщение msg в тему subject.
// Если шина закрыта — ErrClosed. Если темы нет — просто игнорируется.
func (b *broker) Publish(subject string, msg interface{}) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return ErrClosed
	}
	t, ok := b.topics[subject]
	b.mu.RUnlock()
	if !ok {
		return nil
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


// file: topic.go
package subpub

import (
	"sync"
)

const defaultTopicBuf = 128

// topic представляет одну очередь FIFO и набор подписчиков.
type topic struct {
	name         string
	inbox        chan interface{}
	mu           sync.RWMutex
	subs         map[uint64]*subscription
	nextID       uint64
	wg           *sync.WaitGroup
	dispatchOnce sync.Once
	closed       bool
}

func newTopic(name string, wg *sync.WaitGroup) *topic {
	return &topic{
		name:   name,
		inbox:  make(chan interface{}, defaultTopicBuf),
		subs:   make(map[uint64]*subscription),
		nextID: 1,
		wg:     wg,
	}
}

// startDispatcher запускает dispatch только один раз.
func (t *topic) startDispatcher() {
	t.dispatchOnce.Do(func() {
		t.wg.Add(1)
		go t.dispatch()
	})
}

// dispatch читает из inbox и рассылат сообщения подписчикам.
func (t *topic) dispatch() {
	defer t.wg.Done()
	for msg := range t.inbox {
		var blocked []*subscription

		// первый проход: non-blocking
		t.mu.RLock()
		for _, sub := range t.subs {
			select {
			case sub.queue <- msg:
			default:
				blocked = append(blocked, sub)
			}
		}
		t.mu.RUnlock()

		// второй проход: ожидаем только для "тормозов"
		for _, sub := range blocked {
			select {
			case <-sub.unsub:
				// подписчик уже ушёл, пропускаем
			case sub.queue <- msg:
			}
		}
	}

	// после закрытия inbox — закрываем все очереди
	t.mu.RLock()
	for _, sub := range t.subs {
		close(sub.queue)
	}
	t.mu.RUnlock()
}

// addSubscriber добавляет подписчика в тему.
func (t *topic) addSubscriber(s *subscription) {
	t.mu.Lock()
	if t.closed {
		close(s.queue)
		close(s.unsub)
		t.mu.Unlock()
		return
	}
	id := t.nextID
	s.id = id
	t.nextID++
	t.subs[id] = s
	t.mu.Unlock()
	t.startDispatcher()
}

// publish кладёт сообщение в inbox (блокируется, если buf полон).
func (t *topic) publish(msg interface{}) {
	t.mu.RLock()
	if t.closed {
		t.mu.RUnlock()
		return
	}
	t.mu.RUnlock()
	t.inbox <- msg
}

// close закрывает inbox, сигналя dispatcher выйти.
func (t *topic) close() {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.closed = true
	close(t.inbox)
	t.mu.Unlock()
}

// removeSubscriber убирает подписчика по id.
func (t *topic) removeSubscriber(id uint64) {
	t.mu.Lock()
	delete(t.subs, id)
	t.mu.Unlock()
}


// file: subscription.go
package subpub

import "sync"

const defaultSubQueue = 64

// subscription реализует Subscription.
type subscription struct {
	id    uint64
	topic *topic
	user  MessageHandler
	queue chan interface{}
	unsub chan struct{}
	once  sync.Once
}

func newSubscription(cb MessageHandler) *subscription {
	s := &subscription{
		user:  cb,
		queue: make(chan interface{}, defaultSubQueue),
		unsub: make(chan struct{}),
	}
	// горутина-обработчик
	go func() {
		for msg := range s.queue {
			s.user(msg)
		}
	}()
	return s
}

// Unsubscribe удаляет подписчика из темы и очищает ресурсы.
func (s *subscription) Unsubscribe() {
	s.once.Do(func() {
		// сигнал для dispatcher не писать в closed queue
		close(s.unsub)
		// удаляем из темы
		s.topic.removeSubscriber(s.id)
		// закрываем очередь, чтобы завершилась goroutine-handler
		close(s.queue)
	})
}


// file: subpub_test.go
package subpub

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestFIFO(t *testing.T) {
	bus := NewSubPub()
	sub, _ := bus.Subscribe("test", func(msg interface{}) {})

defer sub.Unsubscribe()

	count := 1000
	res := make([]int, 0, count)
	lock := sync.Mutex{}
	sub2, _ := bus.Subscribe("test", func(msg interface{}) {
		lock.Lock()
		res = append(res, msg.(int))
		lock.Unlock()
	})
	defer sub2.Unsubscribe()

	for i := 0; i < count; i++ {
		bus.Publish("test", i)
	}
	// дадим обработчикам время
	time.Sleep(100 * time.Millisecond)

	for i, v := range res {
		if v != i {
			t.Errorf("FIFO broken: at %d got %d", i, v)
			break
		}
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := NewSubPub()

	var wg sync.WaitGroup
	results := make([][]int, 3)

	for i := 0; i < 3; i++ {
		wg.Add(1)
		idx := i
		sub, _ := bus.Subscribe("multi", func(msg interface{}) {
			results[idx] = append(results[idx], msg.(int))
			if len(results[idx]) == 5 {
				wg.Done()
			}
		})
		defer sub.Unsubscribe()
	}

	for i := 0; i < 5; i++ {
		bus.Publish("multi", i)
	}
	// ждём всех
	wg.Wait()

	for i := 0; i < 3; i++ {
		for j, v := range results[i] {
			if v != j {
				t.Errorf("sub[%d] wrong: got %v", i, results[i])
			}
		}
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := NewSubPub()

	ch := make(chan string, 4)
	sub, _ := bus.Subscribe("u", func(msg interface{}) {
		ch <- msg.(string)
	})

	bus.Publish("u", "a")
	time.Sleep(time.Millisecond)
	sub.Unsubscribe()
	bus.Publish("u", "b")
	time.Sleep(time.Millisecond)

	close(ch)
	col := make([]string, 0)
	for v := range ch {
		col = append(col, v)
	}
	if len(col) != 1 || col[0] != "a" {
		t.Errorf("unexpected messages after unsubscribe: %v", col)
	}
}

func TestCloseContext(t *testing.T) {
	bus := NewSubPub()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := bus.Close(ctx)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
