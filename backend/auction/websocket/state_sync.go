package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

// ConnectionState WebSocket 连接状态
type ConnectionState struct {
	ClientID       string    `json:"client_id"`
	AuctionID      int64     `json:"auction_id"`
	UserID         int64     `json:"user_id"`
	ConnectedAt    time.Time `json:"connected_at"`
	LastPongAt     time.Time `json:"last_pong_at"`
	ReconnectCount int       `json:"reconnect_count"`
}

// SyncState 竞拍同步状态
type SyncState struct {
	AuctionID    int64           `json:"auction_id"`
	CurrentPrice decimal.Decimal `json:"current_price"`
	WinnerID     int64           `json:"winner_id"`
	EndTime      time.Time       `json:"end_time"`
	Status       int             `json:"status"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// StateManager 状态管理器
type StateManager struct {
	redis *redis.Client
}

// NewStateManager 创建状态管理器
func NewStateManager(redisClient *redis.Client) *StateManager {
	return &StateManager{
		redis: redisClient,
	}
}

// SaveConnectionState 保存连接状态
func (m *StateManager) SaveConnectionState(ctx context.Context, state *ConnectionState) error {
	key := fmt.Sprintf("conn:state:%s", state.ClientID)
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return m.redis.Set(ctx, key, data, 24*time.Hour).Err()
}

// GetConnectionState 获取连接状态
func (m *StateManager) GetConnectionState(ctx context.Context, clientID string) (*ConnectionState, error) {
	key := fmt.Sprintf("conn:state:%s", clientID)
	data, err := m.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var state ConnectionState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// DeleteConnectionState 删除连接状态
func (m *StateManager) DeleteConnectionState(ctx context.Context, clientID string) error {
	key := fmt.Sprintf("conn:state:%s", clientID)
	return m.redis.Del(ctx, key).Err()
}

// SaveSyncState 保存同步状态
func (m *StateManager) SaveSyncState(ctx context.Context, state *SyncState) error {
	key := fmt.Sprintf("sync:state:%d", state.AuctionID)
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return m.redis.Set(ctx, key, data, 7*24*time.Hour).Err()
}

// GetSyncState 获取同步状态
func (m *StateManager) GetSyncState(ctx context.Context, auctionID int64) (*SyncState, error) {
	key := fmt.Sprintf("sync:state:%d", auctionID)
	data, err := m.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var state SyncState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// UpdateReconnectCount 更新重连次数
func (m *StateManager) UpdateReconnectCount(ctx context.Context, clientID string, count int) error {
	state, err := m.GetConnectionState(ctx, clientID)
	if err != nil {
		return err
	}

	state.ReconnectCount = count
	return m.SaveConnectionState(ctx, state)
}
