// package subpub

// import (
// 	"context"
// 	"sync"
// )

// //---------------------------------------------------------------------------
// // Публичные типы (видит пользователь пакета)
// //---------------------------------------------------------------------------

// // MessageHandler — пользовательский колл-бэк.
// type MessageHandler func(msg interface{})

// // Subscription — то, что возвращает Subscribe().
// type Subscription interface {
// 	Unsubscribe()
// }

// // SubPub — центральный интерфейс шины.
// type SubPub interface {
// 	Subscribe(subject string, cb MessageHandler) (Subscription, error)
// 	Publish(subject string, msg interface{}) error
// 	Close(ctx context.Context) error
// }

// //---------------------------------------------------------------------------
// // Внутренние структуры реализации (скрываем через unexported имена)
// //---------------------------------------------------------------------------

// // topic хранит очередь событий и список подписчиков для одного subject.
// type topic struct {
// 	name string

// 	inbox chan interface{} // <- сюда пишет Publish (одиночный reader = FIFO)

// 	mu       sync.RWMutex             // защищает subs
// 	subs     map[uint64]*subscription // подписчики по уникальному id
// 	nextID   uint64                   // автонумерация подписчиков
// 	dispatch sync.Once                // гарантирует запуск dispatcher

// 	wg     sync.WaitGroup // ждём dispatcher
// 	closed bool           // тема закрыта (после broker.Close, когда subs==0)
// }

// // subscription реализует Subscription.
// type subscription struct {
// 	id   uint64
// 	top  *topic
// 	user MessageHandler // оригинальный MessageHandler

// 	queue  chan interface{} // персональная буферизованная очередь
// 	closed chan struct{}    // сигнал для dispatcher и goroutine-handler
// 	once   sync.Once        // гарантирует однократность Unsubscribe
// }

// //---------------------------------------------------------------------------
// // Типичные константы/настройки (можно вынести в Config)
// //---------------------------------------------------------------------------

// const (
// 	defaultSubQueue = 64  // глубина буфера одного подписчика
// 	defaultTopicBuf = 128 // глубина центральной очереди темы
// )

// //---------------------------------------------------------------------------
// // Пояснения, «зачем» каждое поле
// //---------------------------------------------------------------------------

// /*
// broker
//   mu       — конкурентное изменение карты тем и признака закрытия.
//   topics   — быстро находим topic по subject.
//   closed   — дальнейшие Subscribe/Publish после Close запрещены.
//   wg       — дожидаемся dispatcher-горутин всех topics.

// topic
//   inbox    — одиночная FIFO-очередь; порядком управляет один dispatcher.
//   mu+subs  — под мьютексом добавляем/удаляем подписчиков.
//   nextID   — дешёвый способ дать каждому подписчику id (uint64).
//   dispatch — чтобы dispatcher запускался точно один раз.
//   wg       — ждём, пока dispatcher закончит рассылку и выйдет.
//   closed   — true, когда больше не принимаем Publish'ы.

// subscription
//   queue    — «карман» подписчика; медленный обработчик наполняет
//               только свой буфер, не стопоря других.
//   closed   — закрываем в Unsubscribe, чтобы dispatcher не писал в
//               уже ушедший канал и handler знал, что надо завершиться.
//   once     — защищает от двойного Unsubscribe.

// Потоки данных
// -------------
//  Publish() ---> topic.inbox ----> [dispatcher goroutine] ----> sub.queue
//                                                      (по порядку)
//                                              goroutine(handler) -> cb(msg)

// * dispatcher читает **только** из topic.inbox — один reader = FIFO.
// * Для каждого sub пытается `queue <- msg`.  Если буфер полон —
//   dispatcher **ждёт ТОЛЬКО этого подписчика**, остальные получают msg сразу.
// * queue закрывается в `Unsubscribe()`, тогда dispatcher перестаёт в неё писать,
//   а goroutine-handler выходит из `range queue`.

// Close(ctx)
// ----------
//  1. Под мьютексом ставим broker.closed=true, закрываем все topic.inbox.
//  2. Ждём broker.wg (но прерываемся, если ctx.Done()).
//  3. Дообработавшие goroutine-handler'ы завершаются сами — их ждать не нужно,
//     важно лишь, что мы больше не удерживаем системные ресурсы
//     (каналы, мьютексы, dispatcher-горутину).

// Все эти структуры укладываются в требования:
// * **FIFO** — один dispatcher на тему.
// * **Медленный ≠ тормозит всех** — персональный буфер `queue`.
// * **Unsubscribe** — знает свой `id`, легко удаляется из `topic.subs`.
// * **Нет утечек** — всё что создано, добавляется в соответствующий WaitGroup,
//   всё закрывается/завершается при Unsubscribe или Close.

// */

// func (b *broker) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
// 	topic := t.getOrCreateTopic(subject)
// 	return nil, nil
// }

// func (b *broker) Publish(subject string, msg interface{}) error {
// 	return nil
// }

// func (b *broker) Close(ctx context.Context) error {
// 	return nil
// }

// func NewSubPub() SubPub {
// 	panic("Implement me")
// }

// // func main() {

// // }
