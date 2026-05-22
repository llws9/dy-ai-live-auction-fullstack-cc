package websocket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWebSocketManager_New(t *testing.T) {
	hub := NewHub()
	manager := NewWebSocketManager(hub, nil)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.GetHub())
	assert.Nil(t, manager.GetStateManager())
}

func TestWebSocketManager_BroadcastToRoom(t *testing.T) {
	hub := NewHub()
	manager := NewWebSocketManager(hub, nil)

	go hub.Run()
	defer hub.Stop()

	// 创建房间
	room := NewRoom(1)
	hub.rooms[1] = room
	go room.Run()

	// 广播消息
	msg := &Message{
		Type: "test",
		Data: map[string]interface{}{"content": "hello"},
	}

	// 应该不会 panic
	manager.BroadcastToRoom(1, msg)

	time.Sleep(100 * time.Millisecond)
}

func TestWebSocketManager_StateManager(t *testing.T) {
	// 测试无 Redis 情况
	hub := NewHub()
	manager := NewWebSocketManager(hub, nil)

	// 验证 StateManager 为 nil
	assert.Nil(t, manager.GetStateManager())

	// 但 GetHub 应该正常工作
	assert.NotNil(t, manager.GetHub())
}

func TestWebSocketManager_GetConnectionState(t *testing.T) {
	hub := NewHub()
	manager := NewWebSocketManager(hub, nil)

	// 无 Redis 时，GetConnectionState 应该返回 nil
	ctx := t.Context()
	state, _ := manager.GetConnectionState(ctx, "test-client")

	// 无 Redis，应该返回 nil
	assert.Nil(t, state)
}
