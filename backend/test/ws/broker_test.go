package ws

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 订阅后能收到 publish 的消息
func TestBroker_PubSub(t *testing.T) {
	b := NewBroker(0) // 0 = 不节流，便于断言精确次数

	ch, unsub := b.Subscribe("t1")
	defer unsub()

	go func() {
		b.Publish("t1", Message{Progress: 50, Step: "half"})
		b.Publish("t1", Message{Progress: 100, Step: "done"})
	}()

	got := drain(ch, 2, time.Second)
	require.Len(t, got, 2)
	assert.Equal(t, 50, got[0].Progress)
	assert.Equal(t, 100, got[1].Progress)
}

// 不同 testID 互不干扰
func TestBroker_Isolation(t *testing.T) {
	b := NewBroker(0)
	ch1, u1 := b.Subscribe("t1")
	ch2, u2 := b.Subscribe("t2")
	defer u1()
	defer u2()

	b.Publish("t1", Message{Progress: 10})
	b.Publish("t2", Message{Progress: 20})

	g1 := drain(ch1, 1, 200*time.Millisecond)
	g2 := drain(ch2, 1, 200*time.Millisecond)
	require.Len(t, g1, 1)
	require.Len(t, g2, 1)
	assert.Equal(t, 10, g1[0].Progress)
	assert.Equal(t, 20, g2[0].Progress)
}

// 取消订阅后不再收到消息
func TestBroker_Unsubscribe(t *testing.T) {
	b := NewBroker(0)
	ch, unsub := b.Subscribe("t1")
	unsub()

	b.Publish("t1", Message{Progress: 1})

	select {
	case msg, ok := <-ch:
		if ok {
			t.Fatalf("expected no msg, got %+v", msg)
		}
	case <-time.After(100 * time.Millisecond):
		// ok：通道已关或无人发
	}
}

// 节流：200ms 内多次 publish 仅保留最后一条
func TestBroker_Throttle(t *testing.T) {
	b := NewBroker(100 * time.Millisecond)
	ch, unsub := b.Subscribe("t1")
	defer unsub()

	// 50ms 内发 5 条；节流应合并为 1 条（最后一条）
	for i := 1; i <= 5; i++ {
		b.Publish("t1", Message{Progress: i * 10})
	}

	got := drain(ch, 1, 500*time.Millisecond)
	require.GreaterOrEqual(t, len(got), 1)
	// 最后一次 publish 的 progress=50 必须出现
	last := got[len(got)-1]
	assert.Equal(t, 50, last.Progress)
}

// 慢消费者不阻塞 publisher
func TestBroker_NonBlockingOnSlowConsumer(t *testing.T) {
	b := NewBroker(0)
	_, unsub := b.Subscribe("t1") // 不消费
	defer unsub()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			b.Publish("t1", Message{Progress: i})
		}
		close(done)
	}()

	select {
	case <-done:
		// ok：publish 不被阻塞
	case <-time.After(2 * time.Second):
		t.Fatal("publish blocked by slow consumer")
	}
}

// drain 在 timeout 内尽量收 n 条
func drain(ch <-chan Message, n int, timeout time.Duration) []Message {
	var out []Message
	deadline := time.After(timeout)
	for len(out) < n {
		select {
		case m, ok := <-ch:
			if !ok {
				return out
			}
			out = append(out, m)
		case <-deadline:
			return out
		}
	}
	return out
}

// 并发订阅/取消不应 panic
func TestBroker_ConcurrentSubUnsub(t *testing.T) {
	b := NewBroker(0)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, u := b.Subscribe("t1")
			time.Sleep(time.Millisecond)
			u()
		}()
	}
	wg.Wait()
}
