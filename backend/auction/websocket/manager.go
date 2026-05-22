package websocket

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

// WebSocketManager WebSocket 管理器，统一管理 Hub 和 StateManager
type WebSocketManager struct {
	hub          *Hub
	stateManager *StateManager
	redis        *redis.Client
}

// NewWebSocketManager 创建 WebSocket 管理器
func NewWebSocketManager(hub *Hub, redisClient *redis.Client) *WebSocketManager {
	var stateManager *StateManager
	if redisClient != nil {
		stateManager = NewStateManager(redisClient)
	}

	return &WebSocketManager{
		hub:          hub,
		stateManager: stateManager,
		redis:        redisClient,
	}
}

// GetHub 获取 Hub
func (m *WebSocketManager) GetHub() *Hub {
	return m.hub
}

// GetStateManager 获取状态管理器
func (m *WebSocketManager) GetStateManager() *StateManager {
	return m.stateManager
}

// RegisterClient 注册客户端
func (m *WebSocketManager) RegisterClient(client *Client) {
	// 保存连接状态到 Redis
	if m.stateManager != nil {
		state := &ConnectionState{
			ClientID:       client.ID,
			AuctionID:      client.AuctionID,
			UserID:         client.UserID,
			ConnectedAt:    client.ConnectedAt,
			LastPongAt:     client.ConnectedAt,
			ReconnectCount: 0,
		}

		// 检查是否为重连
		ctx := context.Background()
		existingState, err := m.stateManager.GetConnectionState(ctx, client.ID)
		if err == nil && existingState != nil {
			state.ReconnectCount = existingState.ReconnectCount + 1
			log.Printf("Client reconnecting: client_id=%s, reconnect_count=%d", client.ID, state.ReconnectCount)
		}

		if err := m.stateManager.SaveConnectionState(ctx, state); err != nil {
			log.Printf("Failed to save connection state: %v", err)
		}
	}

	// 注册到 Hub
	m.hub.Register <- client
}

// UnregisterClient 注销客户端
func (m *WebSocketManager) UnregisterClient(client *Client) {
	// 删除连接状态
	if m.stateManager != nil {
		ctx := context.Background()
		if err := m.stateManager.DeleteConnectionState(ctx, client.ID); err != nil {
			log.Printf("Failed to delete connection state: %v", err)
		}
	}

	// 从 Hub 注销
	m.hub.Unregister <- client
}

// BroadcastToRoom 向房间广播消息
func (m *WebSocketManager) BroadcastToRoom(auctionID int64, message *Message) {
	m.hub.BroadcastToRoom(auctionID, message)
}

// GetConnectionState 获取连接状态
func (m *WebSocketManager) GetConnectionState(ctx context.Context, clientID string) (*ConnectionState, error) {
	if m.stateManager == nil {
		return nil, nil
	}
	return m.stateManager.GetConnectionState(ctx, clientID)
}

// RestoreConnection 恢复连接状态
func (m *WebSocketManager) RestoreConnection(client *Client) (*ConnectionState, error) {
	if m.stateManager == nil {
		return nil, nil
	}

	ctx := context.Background()
	state, err := m.stateManager.GetConnectionState(ctx, client.ID)
	if err != nil {
		return nil, err
	}

	// 更新重连次数
	if state != nil {
		state.ReconnectCount++
		state.ConnectedAt = client.ConnectedAt
		if err := m.stateManager.SaveConnectionState(ctx, state); err != nil {
			return nil, err
		}
	}

	return state, nil
}

// Run 启动 Hub
func (m *WebSocketManager) Run() {
	go m.hub.Run()
}

// Stop 停止 Hub
func (m *WebSocketManager) Stop() {
	m.hub.Stop()
}
