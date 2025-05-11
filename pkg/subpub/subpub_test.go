package subpub

import (
	"context"
	"fmt"
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
		fmt.Println("multihandler ", msg.(string))
	})

	bus.Publish("u", "a")
	// time.Sleep(time.Millisecond)
	sub.Unsubscribe()
	bus.Publish("u", "b")
	// time.Sleep(time.Millisecond)

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
