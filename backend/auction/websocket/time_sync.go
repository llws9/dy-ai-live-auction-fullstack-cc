package websocket

import (
	"time"
)

// TimeSyncService 时间同步服务
type TimeSyncService struct {
	serverTimeOffset int64 // 客户端与服务端时间偏差
	hub              *Hub  // WebSocket Hub for broadcasting
}

// NewTimeSyncService 创建时间同步服务
func NewTimeSyncService() *TimeSyncService {
	return &TimeSyncService{}
}

// SetHub 设置 WebSocket Hub
func (s *TimeSyncService) SetHub(hub *Hub) {
	s.hub = hub
}

// GetServerTime 获取服务端时间（毫秒）
func (s *TimeSyncService) GetServerTime() int64 {
	return time.Now().UnixMilli()
}

// CalculateEndTime 计算结束时间（考虑延时）
func (s *TimeSyncService) CalculateEndTime(baseEndTime time.Time, delaySeconds int) time.Time {
	return baseEndTime.Add(time.Duration(delaySeconds) * time.Second)
}

// GetRemainingTime 计算剩余时间（毫秒）
func (s *TimeSyncService) GetRemainingTime(endTime time.Time) int64 {
	remaining := time.Until(endTime)
	if remaining < 0 {
		return 0
	}
	return remaining.Milliseconds()
}

// CreateTimeSyncMessage 创建时间同步消息
func (s *TimeSyncService) CreateTimeSyncMessage(endTime int64) *Message {
	return NewTimeSyncMessage(s.GetServerTime(), endTime)
}

// BroadcastTimeSync 向指定竞拍房间广播时间同步消息
func (s *TimeSyncService) BroadcastTimeSync(auctionID int64, endTime int64) {
	if s.hub == nil {
		return
	}

	msg := s.CreateTimeSyncMessage(endTime)
	s.hub.BroadcastToRoom(auctionID, msg)
}
