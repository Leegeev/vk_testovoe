package subpub

import (
	"sync"

	"github.com/spf13/viper"
)

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
	bufSize := viper.GetInt("bus.topic_buffer_size")
	return &topic{
		name:   name,
		inbox:  make(chan interface{}, bufSize),
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
	for _, sub := range t.subs {
		sub.Unsubscribe()
		// close(sub.queue)
		// close(sub.unsub) // ЭТО Я ДОБАВИЛ
	}
}

// addSubscriber добавляет подписчика в тему.
func (t *topic) addSubscriber(s *subscription) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		close(s.queue)
		close(s.unsub)
		t.mu.Unlock()
		return
	}

	s.id = t.nextID
	s.topic = t
	t.nextID++
	t.subs[s.id] = s

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
