package service

import (
	"sync"
	"time"
)

// ThrottleService 消息节流服务
type ThrottleService struct {
	lastSent     map[string]time.Time
	lastSentLock sync.RWMutex
	interval     time.Duration
}

// NewThrottleService 创建节流服务
func NewThrottleService(interval time.Duration) *ThrottleService {
	return &ThrottleService{
		lastSent: make(map[string]time.Time),
		interval: interval,
	}
}

// ShouldSend 检查是否应该发送消息（节流）
func (s *ThrottleService) ShouldSend(key string) bool {
	s.lastSentLock.Lock()
	defer s.lastSentLock.Unlock()

	now := time.Now()
	lastSent, exists := s.lastSent[key]

	if !exists || now.Sub(lastSent) >= s.interval {
		s.lastSent[key] = now
		return true
	}

	return false
}

// Clear 清除指定 key 的节流状态
func (s *ThrottleService) Clear(key string) {
	s.lastSentLock.Lock()
	defer s.lastSentLock.Unlock()

	delete(s.lastSent, key)
}

// ClearAll 清除所有节流状态
func (s *ThrottleService) ClearAll() {
	s.lastSentLock.Lock()
	defer s.lastSentLock.Unlock()

	s.lastSent = make(map[string]time.Time)
}

// RankingThrottle 排名更新节流器
type RankingThrottle struct {
	throttle *ThrottleService
}

// NewRankingThrottle 创建排名更新节流器
func NewRankingThrottle() *RankingThrottle {
	return &RankingThrottle{
		throttle: NewThrottleService(200 * time.Millisecond),
	}
}

// ShouldSend 检查是否应该发送排名更新
func (t *RankingThrottle) ShouldSend(auctionID int64) bool {
	return t.throttle.ShouldSend(t.getKey(auctionID))
}

func (t *RankingThrottle) getKey(auctionID int64) string {
	return "ranking:" + string(auctionID)
}
