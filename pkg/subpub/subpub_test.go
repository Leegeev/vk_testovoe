package subpub

import (
	"context"
	"go.uber.org/goleak"
	"sync"
	"testing"
	"time"
)

func TestFIFO(t *testing.T) {
	defer goleak.VerifyNone(t)
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

	time.Sleep(100 * time.Millisecond)

	for i, v := range res {
		if v != i {
			t.Errorf("FIFO broken: at %d got %d", i, v)
			break
		}
	}
	ctx, _ := context.WithCancel(context.Background())
	bus.Close(ctx)
}

func TestMultipleSubscribers(t *testing.T) {
	defer goleak.VerifyNone(t)
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
	ctx, _ := context.WithCancel(context.Background())
	bus.Close(ctx)
}

func TestUnsubscribe1(t *testing.T) {
	defer goleak.VerifyNone(t)
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
	ctx, _ := context.WithCancel(context.Background())
	bus.Close(ctx)
}

func TestUnsubscribe2(t *testing.T) {
	defer goleak.VerifyNone(t)
	bus := NewSubPub()

	ch1 := make(chan string, 4)
	ch2 := make(chan string, 4)
	sub1, _ := bus.Subscribe("u", func(msg interface{}) {
		ch1 <- msg.(string)
	})
	sub2, _ := bus.Subscribe("u", func(msg interface{}) {
		ch2 <- msg.(string)
	})
	defer sub2.Unsubscribe()

	bus.Publish("u", "a")
	time.Sleep(time.Millisecond)
	sub1.Unsubscribe()
	bus.Publish("u", "b")
	time.Sleep(time.Millisecond)

	close(ch1)
	close(ch2)
	col1 := make([]string, 0)
	for v := range ch1 {
		col1 = append(col1, v)
	}
	if len(col1) != 1 || col1[0] != "a" {
		t.Errorf("unexpected messages after unsubscribe: %v", col1)
	}
	col2 := make([]string, 0)
	for v := range ch2 {
		col2 = append(col2, v)
	}
	if len(col2) != 2 || col2[0] != "a" || col2[1] != "b" {
		t.Errorf("unexpected messages after unsubscribe: %v", col2)
	}
	ctx, _ := context.WithCancel(context.Background())
	bus.Close(ctx)
}

func TestCloseContext(t *testing.T) {
	defer goleak.VerifyNone(t)
	bus := NewSubPub()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := bus.Close(ctx)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
