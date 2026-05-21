package service

import (
	"context"
	"log"
	"time"
)

// Scheduler 状态转换定时任务
type Scheduler struct {
	auctionService *AuctionService
	ticker         *time.Ticker
	stopChan       chan struct{}
}

// NewScheduler 创建定时任务调度器
func NewScheduler(auctionService *AuctionService) *Scheduler {
	return &Scheduler{
		auctionService: auctionService,
		stopChan:       make(chan struct{}),
	}
}

// Start 启动定时任务
func (s *Scheduler) Start() {
	// 每 1 秒检查一次
	s.ticker = time.NewTicker(1 * time.Second)

	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.checkAuctions()
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
}
