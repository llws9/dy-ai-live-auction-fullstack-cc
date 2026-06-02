package websocket

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newChatHandlerFixture(t *testing.T) (*ChatHandler, *Hub, *miniredis.Miniredis) {
	t.Helper()
	hub := NewHub()
	go hub.Run()

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	filter := NewChatFilter(50, []string{"微信"})
	throttle := NewChatThrottle(rdb, ThrottleConfig{
		UserMax: 1, UserInterval: time.Second,
		RoomMax: 20, RoomInterval: time.Second,
	})
	h := NewChatHandler(hub, filter, throttle)
	return h, hub, mr
}

func dataAs[T any](t *testing.T, m *Message) T {
	t.Helper()
	raw, _ := json.Marshal(m.Data)
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

func TestChatHandler_GuestRejected(t *testing.T) {
	h, hub, _ := newChatHandlerFixture(t)
	defer hub.Stop()

	c := &Client{ID: "guest", UserID: 0, LiveStreamID: 1, Send: make(chan *Message, 4)}
	h.Handle(context.Background(), c, &ChatSendData{LiveStreamID: 1, Text: "hi"})

	select {
	case msg := <-c.Send:
		if msg.Type != MessageTypeError {
			t.Fatalf("expected error message, got %s", msg.Type)
		}
		err := dataAs[ErrorData](t, msg)
		if err.Code != ChatErrCodeNotAuthenticated {
			t.Fatalf("got code %d, want %d", err.Code, ChatErrCodeNotAuthenticated)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("no error reply")
	}
}

func TestChatHandler_QueryUserIDFallbackRejected(t *testing.T) {
	h, hub, _ := newChatHandlerFixture(t)
	defer hub.Stop()

	c := &Client{ID: "spoofed", UserID: 1002, LiveStreamID: 1, Send: make(chan *Message, 4)}
	h.Handle(context.Background(), c, &ChatSendData{LiveStreamID: 1, Text: "spoof"})

	select {
	case msg := <-c.Send:
		if msg.Type != MessageTypeError {
			t.Fatalf("expected error message, got %s", msg.Type)
		}
		err := dataAs[ErrorData](t, msg)
		if err.Code != ChatErrCodeNotAuthenticated {
			t.Fatalf("got code %d, want %d", err.Code, ChatErrCodeNotAuthenticated)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("no error reply")
	}
}

func TestChatHandler_LengthError(t *testing.T) {
	h, hub, _ := newChatHandlerFixture(t)
	defer hub.Stop()

	c := &Client{ID: "u1", UserID: 1, Authenticated: true, LiveStreamID: 1, Send: make(chan *Message, 4)}
	h.Handle(context.Background(), c, &ChatSendData{LiveStreamID: 1, Text: ""})

	msg := <-c.Send
	if dataAs[ErrorData](t, msg).Code != ChatErrCodeLengthExceeded {
		t.Fatal("expected length error")
	}
}

func TestChatHandler_BlockedWord(t *testing.T) {
	h, hub, _ := newChatHandlerFixture(t)
	defer hub.Stop()

	c := &Client{ID: "u2", UserID: 2, Authenticated: true, LiveStreamID: 1, Send: make(chan *Message, 4)}
	h.Handle(context.Background(), c, &ChatSendData{LiveStreamID: 1, Text: "加我微信"})

	msg := <-c.Send
	if dataAs[ErrorData](t, msg).Code != ChatErrCodeBlockedWord {
		t.Fatal("expected blocked-word error")
	}
}

func TestChatHandler_HappyPath(t *testing.T) {
	h, hub, _ := newChatHandlerFixture(t)
	defer hub.Stop()

	// 先把客户端注册进直播间，才能收到广播
	c := &Client{ID: "u3", UserID: 3, Authenticated: true, UserName: "Alice", LiveStreamID: 7, Send: make(chan *Message, 4)}
	hub.RegisterToLiveStream(c)
	time.Sleep(20 * time.Millisecond)

	h.Handle(context.Background(), c, &ChatSendData{LiveStreamID: 7, Text: "hello", ClientMsgID: "cm-1"})

	timeout := time.After(time.Second)
	for {
		select {
		case msg := <-c.Send:
			if msg.Type == MessageTypeChatMessage {
				d := dataAs[ChatMessageData](t, msg)
				if d.UserID != 3 || d.UserName != "Alice" || d.Text != "hello" || d.ClientMsgID != "cm-1" {
					t.Fatalf("unexpected payload: %+v", d)
				}
				return
			}
		case <-timeout:
			t.Fatal("did not receive chat_message")
		}
	}
}

func TestChatHandler_RateLimit(t *testing.T) {
	h, hub, _ := newChatHandlerFixture(t)
	defer hub.Stop()

	c := &Client{ID: "u4", UserID: 4, Authenticated: true, LiveStreamID: 7, Send: make(chan *Message, 8)}
	hub.RegisterToLiveStream(c)
	time.Sleep(20 * time.Millisecond)

	h.Handle(context.Background(), c, &ChatSendData{LiveStreamID: 7, Text: "first"})
	h.Handle(context.Background(), c, &ChatSendData{LiveStreamID: 7, Text: "second"})

	rateLimitedSeen := false
	deadline := time.After(time.Second)
	for !rateLimitedSeen {
		select {
		case msg := <-c.Send:
			if msg.Type == MessageTypeError && dataAs[ErrorData](t, msg).Code == ChatErrCodeRateLimited {
				rateLimitedSeen = true
			}
		case <-deadline:
			t.Fatal("did not see rate-limited error")
		}
	}
}
