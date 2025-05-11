package subpub

import "sync"

type subscription struct {
	id   uint64
	top  *topic
	user MessageHandler // оригинальный MessageHandler

	queue  chan interface{} // персональная буферизованная очередь
	closed chan struct{}    // сигнал для dispatcher и goroutine-handler
	once   sync.Once        // гарантирует однократность Unsubscribe
}

func newSubscription(cb MessageHandler) *subscription {
	s := &subscription{
		user:   cb,
		queue:  make(chan interface{}, defaultSubQueue),
		closed: make(chan struct{}),
	}

	go func() {
		for msg := range s.queue {
			s.user(msg)
		}
	}()

	return s
}

func (s *subscription) Unsubscribe() {
	s.once.Do(func() {
		close(s.closed)
		s.top.removeSubscriber(s.id)
		close(s.queue)
	})
}
