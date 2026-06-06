package service

import (
	"context"
	"log"
	"time"

	"auction-service/websocket"
)

// Scheduler 状态转换定时任务
type Scheduler struct {
	auctionService  *AuctionService
	timeSyncService *websocket.TimeSyncService
	hub             *websocket.Hub
	ticker          *time.Ticker
	timeSyncTicker  *time.Ticker
	stopChan        chan struct{}
}

// NewScheduler 创建定时任务调度器
func NewScheduler(auctionService *AuctionService) *Scheduler {
	return &Scheduler{
		auctionService:  auctionService,
		timeSyncService: websocket.NewTimeSyncService(),
		stopChan:        make(chan struct{}),
	}
}

// SetHub 设置 WebSocket Hub
func (s *Scheduler) SetHub(hub *websocket.Hub) {
	s.hub = hub
	s.timeSyncService.SetHub(hub)
}

// Start 启动定时任务
func (s *Scheduler) Start() {
	// 每 1 秒检查一次竞拍状态
	s.ticker = time.NewTicker(1 * time.Second)

	// 每 5 秒推送时间同步
	s.timeSyncTicker = time.NewTicker(5 * time.Second)

	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.checkAuctions()
			case <-s.timeSyncTicker.C:
				s.broadcastTimeSync()
			case <-s.stopChan:
				return
			}
		}
	}()

	log.Println("Auction scheduler started")
}

// Stop 停止定时任务
func (s *Scheduler) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	if s.timeSyncTicker != nil {
		s.timeSyncTicker.Stop()
	}
	close(s.stopChan)
	log.Println("Auction scheduler stopped")
}

// checkAuctions 检查竞拍状态
func (s *Scheduler) checkAuctions() {
	ctx := context.Background()

	// 检查应该开始的竞拍
	if err := s.auctionService.CheckAndStartAuctions(ctx); err != nil {
		log.Printf("Error checking auctions to start: %v", err)
	}

	// 检查应该结束的竞拍
	if err := s.auctionService.CheckAndEndAuctions(ctx); err != nil {
		log.Printf("Error checking auctions to end: %v", err)
	}

	if err := s.auctionService.RetryUnfinishedSettlements(ctx, 100); err != nil {
		log.Printf("Error retrying unfinished auction settlements: %v", err)
	}
}

// broadcastTimeSync 广播时间同步消息
func (s *Scheduler) broadcastTimeSync() {
	if s.hub == nil {
		return
	}

	ctx := context.Background()

	// 获取进行中的竞拍
	auctions, err := s.auctionService.GetAuctionsByStatus(ctx, 1) // status=1 进行中
	if err != nil {
		log.Printf("Error getting ongoing auctions for time sync: %v", err)
		return
	}

	// 向每个竞拍房间广播时间同步
	for _, auction := range auctions {
		s.timeSyncService.BroadcastTimeSync(auction.ID, auction.EndTime.UnixMilli())
	}
}
