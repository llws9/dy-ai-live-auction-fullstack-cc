package websocket

import (
	"testing"
	"time"
)

type recordingPresenceCountSink struct {
	updates []int
}

func (s *recordingPresenceCountSink) SetLiveViewerCount(liveStreamID int64, count int) error {
	s.updates = append(s.updates, count)
	return nil
}

func newTestRoomClient(id string) *Client {
	return &Client{
		ID:   id,
		Send: make(chan *Message, 16),
	}
}

func newPresenceRoomClient(id string, userID int64, userName string, authenticated bool) *Client {
	return &Client{
		ID:            id,
		LiveStreamID:  777,
		UserID:        userID,
		UserName:      userName,
		Authenticated: authenticated,
		Send:          make(chan *Message, 16),
	}
}

func nextPresenceUpdate(t *testing.T, c *Client) *LivePresenceUpdateData {
	t.Helper()
	select {
	case msg := <-c.Send:
		if msg.Type != MessageTypeLivePresenceUpdate {
			t.Fatalf("got message type %s, want %s", msg.Type, MessageTypeLivePresenceUpdate)
		}
		data, ok := msg.Data.(*LivePresenceUpdateData)
		if !ok {
			t.Fatalf("got presence data %T, want *LivePresenceUpdateData", msg.Data)
		}
		return data
	case <-time.After(time.Second):
		t.Fatal("did not receive live presence update within 1s")
		return nil
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

func TestHub_RegisterToLiveStreamIsImmediatelyVisible(t *testing.T) {
	h := NewHub()
	defer h.Stop()

	c := &Client{ID: "instant", LiveStreamID: 321, Send: make(chan *Message, 4)}
	h.RegisterToLiveStream(c)

	room := h.GetLiveStreamRoom(321)
	if room == nil {
		t.Fatal("live stream room was not created")
	}
	if got := room.GetClientCount(); got != 1 {
		t.Fatalf("client count = %d, want 1", got)
	}
}

func TestHub_UnregisterFromLiveStreamDeletesEmptyRoom(t *testing.T) {
	h := NewHub()
	defer h.Stop()

	c := &Client{ID: "last", LiveStreamID: 654, Send: make(chan *Message, 4)}
	h.RegisterToLiveStream(c)
	h.UnregisterFromLiveStream(c)

	if room := h.GetLiveStreamRoom(654); room != nil {
		t.Fatal("empty live stream room should be deleted")
	}
}

func TestLiveStreamRoom_PresenceDeduplicatesUserConnections(t *testing.T) {
	r := NewLiveStreamRoom(777)
	defer r.Close()

	first := newPresenceRoomClient("c1", 42, "张三", true)
	second := newPresenceRoomClient("c2", 42, "张三", true)

	r.registerClient(first)
	firstSnapshot := nextPresenceUpdate(t, first)
	if firstSnapshot.ViewerCount != 1 {
		t.Fatalf("first snapshot viewer_count = %d, want 1", firstSnapshot.ViewerCount)
	}

	r.registerClient(second)
	secondSnapshot := nextPresenceUpdate(t, second)
	if secondSnapshot.ViewerCount != 1 {
		t.Fatalf("second snapshot viewer_count = %d, want deduplicated 1", secondSnapshot.ViewerCount)
	}
	if len(secondSnapshot.Viewers) != 1 || secondSnapshot.Viewers[0].UserID != 42 {
		t.Fatalf("viewers = %#v, want one viewer for user 42", secondSnapshot.Viewers)
	}
}

func TestLiveStreamRoom_PresenceSyncsDeduplicatedViewerCount(t *testing.T) {
	sink := &recordingPresenceCountSink{}
	r := NewLiveStreamRoomWithPresenceCountSink(777, sink)
	defer r.Close()

	first := newPresenceRoomClient("c1", 42, "张三", true)
	second := newPresenceRoomClient("c2", 42, "张三", true)

	r.registerClient(first)
	_ = nextPresenceUpdate(t, first)
	r.registerClient(second)
	_ = nextPresenceUpdate(t, second)
	r.unregisterClient(first)
	_ = nextPresenceUpdate(t, second)
	r.unregisterClient(second)

	want := []int{1, 1, 1, 0}
	if len(sink.updates) != len(want) {
		t.Fatalf("sink updates = %#v, want %#v", sink.updates, want)
	}
	for i := range want {
		if sink.updates[i] != want[i] {
			t.Fatalf("sink updates = %#v, want %#v", sink.updates, want)
		}
	}
}

func TestLiveStreamRoom_PresenceRemovesUserAfterLastConnectionLeaves(t *testing.T) {
	r := NewLiveStreamRoom(777)
	defer r.Close()

	first := newPresenceRoomClient("c1", 42, "张三", true)
	second := newPresenceRoomClient("c2", 42, "张三", true)
	r.registerClient(first)
	_ = nextPresenceUpdate(t, first)
	r.registerClient(second)
	_ = nextPresenceUpdate(t, second)

	r.unregisterClient(first)
	afterFirstLeave := nextPresenceUpdate(t, second)
	if afterFirstLeave.ViewerCount != 1 {
		t.Fatalf("viewer_count after first client leaves = %d, want 1", afterFirstLeave.ViewerCount)
	}

	r.unregisterClient(second)
	if got := r.GetPresenceSnapshot().ViewerCount; got != 0 {
		t.Fatalf("viewer_count after last client leaves = %d, want 0", got)
	}
}

func TestLiveStreamRoom_PresenceUpdateDoesNotEnterHistory(t *testing.T) {
	r := NewLiveStreamRoom(777)
	defer r.Close()

	c := newPresenceRoomClient("c1", 42, "张三", true)
	r.registerClient(c)
	_ = nextPresenceUpdate(t, c)

	if history := r.GetHistory(); len(history) != 0 {
		t.Fatalf("history len = %d, want 0 because presence is ephemeral", len(history))
	}
}

func TestLiveStreamRoom_UnauthenticatedClientDoesNotAppearInPresenceViewers(t *testing.T) {
	r := NewLiveStreamRoom(777)
	defer r.Close()

	c := newPresenceRoomClient("legacy-query-user", 99, "伪造用户", false)
	r.registerClient(c)
	snapshot := r.GetPresenceSnapshot()

	if snapshot.ViewerCount != 0 {
		t.Fatalf("viewer_count = %d, want 0 for unauthenticated client", snapshot.ViewerCount)
	}
	if len(snapshot.Viewers) != 0 {
		t.Fatalf("viewers = %#v, want empty for unauthenticated client", snapshot.Viewers)
	}
	select {
	case msg := <-c.Send:
		t.Fatalf("unauthenticated client should not receive presence update, got %s", msg.Type)
	default:
	}
}

func TestLiveStreamRoom_UnauthenticatedClientDoesNotReceiveAuthenticatedPresenceViewers(t *testing.T) {
	r := NewLiveStreamRoom(777)
	defer r.Close()

	legacy := newPresenceRoomClient("legacy-query-user", 99, "伪造用户", false)
	authenticated := newPresenceRoomClient("jwt-user", 42, "张三", true)
	r.registerClient(legacy)
	r.registerClient(authenticated)
	_ = nextPresenceUpdate(t, authenticated)

	select {
	case msg := <-legacy.Send:
		t.Fatalf("unauthenticated client should not receive authenticated presence update, got %#v", msg.Data)
	default:
	}
}
