package service

import (
	"context"
	"log"
	"time"

	"auction-service/model"
	"auction-service/websocket"
)

// Scheduler 状态转换定时任务
type Scheduler struct {
	auctionService   *AuctionService
	timeSyncService  *websocket.TimeSyncService
	hub              *websocket.Hub
	checkInterval    time.Duration
	timeSyncInterval time.Duration
	ticker           *time.Ticker
	timeSyncTicker   *time.Ticker
	stopChan         chan struct{}
}

const (
	defaultAuctionCheckInterval = 200 * time.Millisecond
	defaultTimeSyncInterval     = 5 * time.Second
)

// NewScheduler 创建定时任务调度器
func NewScheduler(auctionService *AuctionService) *Scheduler {
	return &Scheduler{
		auctionService:   auctionService,
		timeSyncService:  websocket.NewTimeSyncService(),
		checkInterval:    defaultAuctionCheckInterval,
		timeSyncInterval: defaultTimeSyncInterval,
		stopChan:         make(chan struct{}),
	}
}

// SetHub 设置 WebSocket Hub
func (s *Scheduler) SetHub(hub *websocket.Hub) {
	s.hub = hub
	s.timeSyncService.SetHub(hub)
}

// Start 启动定时任务
func (s *Scheduler) Start() {
	// 高频检查竞拍状态，降低 end_time 到结算通知之间的体感延迟。
	s.ticker = time.NewTicker(s.checkInterval)

	// 每 5 秒推送时间同步
	s.timeSyncTicker = time.NewTicker(s.timeSyncInterval)

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
	if err := s.checkAndEndAuctions(ctx); err != nil {
		log.Printf("Error checking auctions to end: %v", err)
	}

	if err := s.auctionService.RetryUnfinishedSettlements(ctx, 100); err != nil {
		log.Printf("Error retrying unfinished auction settlements: %v", err)
	}
}

func (s *Scheduler) checkAndEndAuctions(ctx context.Context) error {
	now := auctionBusinessNow()
	auctions, err := s.auctionService.auctionDAO.GetExpiredAuctions(ctx, now)
	if err != nil {
		return err
	}

	for _, auction := range auctions {
		if err := s.auctionService.EndAuction(ctx, auction.ID); err != nil {
			// 记录错误但继续处理其他竞拍
			continue
		}
		s.broadcastAuctionEnded(ctx, auction.ID)
	}

	return nil
}

func (s *Scheduler) broadcastAuctionEnded(ctx context.Context, auctionID int64) {
	if s.hub == nil {
		return
	}

	auction, err := s.auctionService.GetAuction(ctx, auctionID)
	if err != nil || auction == nil {
		return
	}

	winnerID := int64(0)
	if auction.WinnerID != nil {
		winnerID = *auction.WinnerID
	}

	s.hub.BroadcastToRoom(auction.ID, websocket.NewAuctionEndedMessage(&websocket.AuctionEndedData{
		AuctionID:  auction.ID,
		WinnerID:   winnerID,
		FinalPrice: auction.CurrentPrice,
		EndTime:    auction.EndTime.UnixMilli(),
	}))
}

// broadcastTimeSync 广播时间同步消息（覆盖进行中 + 延时中的竞拍）
func (s *Scheduler) broadcastTimeSync() {
	if s.hub == nil {
		return
	}

	ctx := context.Background()

	statuses := []model.AuctionStatus{
		model.AuctionStatusOngoing,
		model.AuctionStatusDelayed,
	}
	for _, status := range statuses {
		auctions, err := s.auctionService.GetAuctionsByStatus(ctx, int(status))
		if err != nil {
			log.Printf("Error getting auctions(status=%d) for time sync: %v", status, err)
			continue
		}
		for _, auction := range auctions {
			s.timeSyncService.BroadcastTimeSync(auction.ID, auction.EndTime.UnixMilli())
		}
	}
}
