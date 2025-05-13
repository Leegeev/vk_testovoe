package subpub

import (
	"sync"

	"github.com/spf13/viper"
)

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
	bufSize := viper.GetInt("bus.subscriber_queue_size")
	s := &subscription{
		user:  cb,
		queue: make(chan interface{}, bufSize),
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
