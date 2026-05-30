package ws

import (
	"sync"
	"sync/atomic"
	"time"
)

// Message WS 推送的进度消息（也用于 broker 内部）
type Message struct {
	TestID   string         `json:"test_id"`
	Progress int            `json:"progress"`
	Step     string         `json:"step"`
	Metrics  map[string]any `json:"metrics,omitempty"`
	TS       int64          `json:"ts"`
}

// Broker 按 test_id 分发进度消息，支持服务端节流
//
// 设计要点：
//  1. 每个订阅者一个有缓冲 channel + 一个 dropper goroutine，慢消费者最多丢消息，
//     不阻塞 publisher。
//  2. 节流（throttle > 0）：每个订阅者维护一份"待发送"快照，节流定时器到期时
//     合并发送最新一条。throttle == 0 表示直通。
type Broker struct {
	throttle time.Duration
	mu       sync.RWMutex
	subs     map[string]map[*subscriber]struct{} // testID → subs
}

// NewBroker 创建 Broker。throttle <= 0 表示不节流。
func NewBroker(throttle time.Duration) *Broker {
	return &Broker{
		throttle: throttle,
		subs:     make(map[string]map[*subscriber]struct{}),
	}
}

type subscriber struct {
	out chan Message

	// 节流相关（只在 throttle>0 时使用）
	throttle time.Duration
	mu       sync.Mutex
	pending  *Message // 最新待发送
	timer    *time.Timer
	closed   atomic.Bool
}

// Subscribe 订阅指定 testID，返回接收通道与 unsubscribe 函数。
func (b *Broker) Subscribe(testID string) (<-chan Message, func()) {
	s := &subscriber{
		out:      make(chan Message, 16),
		throttle: b.throttle,
	}

	b.mu.Lock()
	set, ok := b.subs[testID]
	if !ok {
		set = make(map[*subscriber]struct{})
		b.subs[testID] = set
	}
	set[s] = struct{}{}
	b.mu.Unlock()

	unsub := func() {
		if !s.closed.CompareAndSwap(false, true) {
			return
		}
		b.mu.Lock()
		if set, ok := b.subs[testID]; ok {
			delete(set, s)
			if len(set) == 0 {
				delete(b.subs, testID)
			}
		}
		b.mu.Unlock()

		s.mu.Lock()
		if s.timer != nil {
			s.timer.Stop()
		}
		s.mu.Unlock()
		close(s.out)
	}
	return s.out, unsub
}

// Publish 向订阅了 testID 的所有 sub 投递消息（节流 / 非阻塞）。
func (b *Broker) Publish(testID string, m Message) {
	m.TestID = testID
	if m.TS == 0 {
		m.TS = time.Now().UnixMilli()
	}

	b.mu.RLock()
	set := b.subs[testID]
	subs := make([]*subscriber, 0, len(set))
	for s := range set {
		subs = append(subs, s)
	}
	b.mu.RUnlock()

	for _, s := range subs {
		s.deliver(m)
	}
}

// deliver 将一条消息送达 subscriber，按节流策略合并。
func (s *subscriber) deliver(m Message) {
	if s.closed.Load() {
		return
	}

	if s.throttle <= 0 {
		nonBlockingSend(s.out, m)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 把最新一条覆盖到 pending，定时器到期统一发送
	cp := m
	s.pending = &cp
	if s.timer == nil {
		s.timer = time.AfterFunc(s.throttle, s.flush)
	}
}

func (s *subscriber) flush() {
	s.mu.Lock()
	msg := s.pending
	s.pending = nil
	s.timer = nil
	s.mu.Unlock()

	if msg == nil || s.closed.Load() {
		return
	}
	nonBlockingSend(s.out, *msg)
}

// nonBlockingSend 非阻塞发送：满了就丢弃最旧的一条再尝试一次。
func nonBlockingSend(ch chan Message, m Message) {
	select {
	case ch <- m:
		return
	default:
	}
	// 通道满：尝试腾出最旧一条
	select {
	case <-ch:
	default:
	}
	select {
	case ch <- m:
	default:
	}
}
