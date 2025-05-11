package subpub

import "sync"

type topic struct {
	name string

	inbox chan interface{} // <- сюда пишет Publish (одиночный reader = FIFO)

	mu       sync.RWMutex             // защищает subs
	subs     map[uint64]*subscription // подписчики по уникальному id
	nextID   uint64                   // автонумерация подписчиков
	dispatch sync.Once                // гарантирует запуск dispatcher

	wg     sync.WaitGroup // ждём dispatcher
	closed bool           // тема закрыта (после broker.Close, когда subs==0)
}

func newTopic(name string, wg sync.WaitGroup) *topic {
	return &topic{
		name:   name,
		inbox:  make(chan interface{}),
		subs:   make(map[uint64]*subscription),
		nextID: 1,
		wg:     wg,
	}
}

func (t *topic) addSubscriber(sub *subscription) {
	t.mu.Lock()
	if t.closed {

	}
}

func (t *topic) removeSubscriber(subID uint64) {

}

//
