package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"auction-service/dao"
	"auction-service/model"
	"auction-service/websocket"

	"github.com/shopspring/decimal"
)

// AuctionService 竞拍服务
type AuctionService struct {
	auctionDAO         *dao.AuctionDAO
	bidDAO             *dao.BidDAO
	notificationSender NotificationSender
	stateManager       *websocket.StateManager
}

// NewAuctionService 创建竞拍服务
func NewAuctionService(auctionDAO *dao.AuctionDAO) *AuctionService {
	return &AuctionService{
		auctionDAO: auctionDAO,
	}
}

// SetBidDAO 设置出价DAO
func (s *AuctionService) SetBidDAO(bidDAO *dao.BidDAO) {
	s.bidDAO = bidDAO
}

// SetNotificationSender 设置通知发送服务
func (s *AuctionService) SetNotificationSender(sender NotificationSender) {
	s.notificationSender = sender
}

// SetStateManager 设置 WebSocket 同步状态管理器。
func (s *AuctionService) SetStateManager(stateManager *websocket.StateManager) {
	s.stateManager = stateManager
}

// SetSkyLampDAO 设置点天灯DAO（用于更新统计数据）
func (s *AuctionService) SetSkyLampDAO(skyLampDAO *dao.SkyLampDAO) {
	// AuctionService暂时不需要SkyLampDAO
	// 保留此方法以备将来扩展
}

// CreateAuctionRequest 创建竞拍请求
type CreateAuctionRequest struct {
	ProductID int64
	StartTime time.Time
	EndTime   time.Time
}

// CreateAuction 创建竞拍
func (s *AuctionService) CreateAuction(ctx context.Context, req *CreateAuctionRequest) (*model.Auction, error) {
	if req.EndTime.Before(req.StartTime) {
		return nil, errors.New("结束时间不能早于开始时间")
	}

	auction := &model.Auction{
		ProductID:    req.ProductID,
		Status:       model.AuctionStatusPending,
		CurrentPrice: decimal.Zero,
		StartTime:    req.StartTime,
		EndTime:      req.EndTime,
		DelayUsed:    0,
	}

	if err := s.auctionDAO.Create(ctx, auction); err != nil {
		return nil, err
	}

	return auction, nil
}

// GetAuction 获取竞拍详情
func (s *AuctionService) GetAuction(ctx context.Context, id int64) (*model.Auction, error) {
	return s.auctionDAO.GetByID(ctx, id)
}

// CancelAuction 取消竞拍
func (s *AuctionService) CancelAuction(ctx context.Context, id int64) error {
	auction, err := s.auctionDAO.GetByID(ctx, id)
	if err != nil {
		return err
	}

	sm := NewStateMachine(auction)
	if !sm.CanCancel() {
		return errors.New("当前状态无法取消")
	}

	if err := sm.Transition(model.AuctionStatusCancelled); err != nil {
		return err
	}

	if err := s.auctionDAO.Update(ctx, auction); err != nil {
		return err
	}
	s.saveSyncState(ctx, auction)
	return nil
}

// StartAuction 开始竞拍
func (s *AuctionService) StartAuction(ctx context.Context, id int64) error {
	auction, err := s.auctionDAO.GetByID(ctx, id)
	if err != nil {
		return err
	}

	sm := NewStateMachine(auction)
	if !sm.CanStart() {
		return errors.New("当前状态无法开始")
	}

	if err := sm.Transition(model.AuctionStatusOngoing); err != nil {
		return err
	}

	if err := s.auctionDAO.Update(ctx, auction); err != nil {
		return err
	}
	s.saveSyncState(ctx, auction)
	return nil
}

// EndAuction 结束竞拍
func (s *AuctionService) EndAuction(ctx context.Context, id int64) error {
	auction, err := s.auctionDAO.GetByID(ctx, id)
	if err != nil {
		return err
	}

	sm := NewStateMachine(auction)
	if err := sm.Transition(model.AuctionStatusEnded); err != nil {
		return err
	}

	if err := s.auctionDAO.Update(ctx, auction); err != nil {
		return err
	}
	s.saveSyncState(ctx, auction)

	// 发送竞拍结果通知
	s.sendAuctionResultNotifications(ctx, auction)

	return nil
}

func (s *AuctionService) saveSyncState(ctx context.Context, auction *model.Auction) {
	_ = SaveAuctionSyncState(ctx, s.stateManager, auction)
}

// sendAuctionResultNotifications 发送竞拍结果通知
func (s *AuctionService) sendAuctionResultNotifications(ctx context.Context, auction *model.Auction) {
	if s.notificationSender == nil || s.bidDAO == nil {
		return
	}

	// 获取所有出价者
	bids, err := s.bidDAO.GetRanking(ctx, auction.ID, 1000)
	if err != nil {
		return
	}

	if len(bids) == 0 {
		return // 无人出价，无需通知
	}

	// 中标者（第一个）
	var winnerID int64
	if auction.WinnerID != nil && *auction.WinnerID > 0 {
		winnerID = *auction.WinnerID
	} else if len(bids) > 0 {
		winnerID = bids[0].UserID
	}
	finalPrice := auction.CurrentPrice

	// 发送中标通知
	go func() {
		_ = s.notificationSender.SendNotification(ctx, &model.NotificationRequest{
			UserID:  winnerID,
			Type:    model.NotificationTypeAuctionWon,
			Title:   "竞拍中标",
			Content: fmt.Sprintf("恭喜！您以 %s 元中标了竞拍", finalPrice.StringFixed(2)),
			Data: map[string]interface{}{
				"auction_id":  auction.ID,
				"final_price": finalPrice.StringFixed(2),
			},
		})
	}()

	// 发送未中标通知给其他参与者
	var loserRequests []*model.NotificationRequest
	for _, bid := range bids {
		if bid.UserID == winnerID {
			continue // 跳过中标者
		}
		loserRequests = append(loserRequests, &model.NotificationRequest{
			UserID:  bid.UserID,
			Type:    model.NotificationTypeAuctionLost,
			Title:   "竞拍未中标",
			Content: fmt.Sprintf("很遗憾，您未能中标。最终成交价为 %s 元", finalPrice.StringFixed(2)),
			Data: map[string]interface{}{
				"auction_id":   auction.ID,
				"winner_price": finalPrice.StringFixed(2),
			},
		})
	}

	// 批量发送未中标通知
	if len(loserRequests) > 0 {
		go func() {
			_ = s.notificationSender.SendBatchNotifications(ctx, loserRequests)
		}()
	}
}

// CheckAndStartAuctions 检查并开始应该开始的竞拍
func (s *AuctionService) CheckAndStartAuctions(ctx context.Context) error {
	auctions, err := s.auctionDAO.GetPendingAuctionsToStart(ctx)
	if err != nil {
		return err
	}

	for _, auction := range auctions {
		if err := s.StartAuction(ctx, auction.ID); err != nil {
			// 记录错误但继续处理其他竞拍
			continue
		}
	}

	return nil
}

// CheckAndEndAuctions 检查并结束应该结束的竞拍
func (s *AuctionService) CheckAndEndAuctions(ctx context.Context) error {
	auctions, err := s.auctionDAO.GetExpiredAuctions(ctx)
	if err != nil {
		return err
	}

	for _, auction := range auctions {
		if err := s.EndAuction(ctx, auction.ID); err != nil {
			// 记录错误但继续处理其他竞拍
			continue
		}
	}

	return nil
}

// IsAuctionActive 检查竞拍是否活跃
func (s *AuctionService) IsAuctionActive(ctx context.Context, id int64) (bool, error) {
	auction, err := s.auctionDAO.GetByID(ctx, id)
	if err != nil {
		return false, err
	}

	sm := NewStateMachine(auction)
	return sm.CanBid(), nil
}

// ListAuctions 获取竞拍列表
func (s *AuctionService) ListAuctions(ctx context.Context, status *model.AuctionStatus, page, pageSize int) ([]model.Auction, int64, error) {
	return s.auctionDAO.List(ctx, status, page, pageSize)
}

// ListAllAuctions 获取所有竞拍（不分页）
func (s *AuctionService) ListAllAuctions(ctx context.Context) ([]model.Auction, error) {
	return s.auctionDAO.ListAll(ctx)
}

// GetAuctionsByStatus 根据状态获取竞拍列表
func (s *AuctionService) GetAuctionsByStatus(ctx context.Context, status int) ([]model.Auction, error) {
	slice, _, err := s.auctionDAO.List(ctx, (*model.AuctionStatus)(&status), 1, 1000)
	return slice, err
}

// ListAuctionsWithFilters 获取竞拍列表（支持多条件筛选）
func (s *AuctionService) ListAuctionsWithFilters(ctx context.Context, filters *dao.AuctionFilters, page, pageSize int) ([]model.Auction, int64, error) {
	return s.auctionDAO.ListWithFilters(ctx, filters, page, pageSize)
}

// GetAuctionBids 获取竞拍的出价记录
func (s *AuctionService) GetAuctionBids(ctx context.Context, auctionID int64, limit int) ([]model.Bid, error) {
	if s.bidDAO == nil {
		return nil, errors.New("bidDAO not initialized")
	}
	return s.bidDAO.ListByAuctionID(ctx, auctionID, limit)
}
