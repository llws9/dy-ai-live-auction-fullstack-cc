package websocket

import (
	"testing"
	"time"
)

func newTestRoomClient(id string) *Client {
	return &Client{
		ID:   id,
		Send: make(chan *Message, 16),
	}
}

func TestLiveStreamRoom_BroadcastDelivers(t *testing.T) {
	r := NewLiveStreamRoom(123)
	go r.Run()
	defer r.Close()

	c := newTestRoomClient("c1")
	r.Register <- c
	time.Sleep(20 * time.Millisecond)

	msg := NewChatMessage(&ChatMessageData{LiveStreamID: 123, Text: "hello"})
	r.Broadcast <- msg

	select {
	case got := <-c.Send:
		if got.Type != MessageTypeChatMessage {
			t.Fatalf("got type %s, want chat_message", got.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive broadcast within 1s")
	}
}

func TestLiveStreamRoom_RingBufferReplay(t *testing.T) {
	r := NewLiveStreamRoom(456)
	go r.Run()
	defer r.Close()

	// 先广播 105 条（环形缓冲容量 100）
	for i := 0; i < 105; i++ {
		r.Broadcast <- NewChatMessage(&ChatMessageData{LiveStreamID: 456, Text: "x"})
	}
	time.Sleep(50 * time.Millisecond)

	history := r.GetHistory()
	if len(history) != 100 {
		t.Fatalf("history len = %d, want 100", len(history))
	}
}

func TestLiveStreamRoom_RegisterReplaysHistory(t *testing.T) {
	r := NewLiveStreamRoom(789)
	go r.Run()
	defer r.Close()

	// 注入 3 条历史
	for i := 0; i < 3; i++ {
		r.Broadcast <- NewChatMessage(&ChatMessageData{LiveStreamID: 789, Text: "old"})
	}
	time.Sleep(20 * time.Millisecond)

	c := newTestRoomClient("late")
	r.Register <- c
	time.Sleep(20 * time.Millisecond)

	// 进房后应立即收到 3 条历史
	got := 0
	timeout := time.After(500 * time.Millisecond)
loop:
	for {
		select {
		case <-c.Send:
			got++
			if got == 3 {
				break loop
			}
		case <-timeout:
			break loop
		}
	}
	if got != 3 {
		t.Fatalf("replayed %d messages, want 3", got)
	}
}
