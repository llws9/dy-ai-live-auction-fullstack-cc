package service

import (
	"context"
	"errors"
	"time"

	"auction-service/dao"
	"auction-service/model"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// AuctionService 竞拍服务
type AuctionService struct {
	auctionDAO        *dao.AuctionDAO
	bidDAO            *dao.BidDAO
	settlementService *AuctionSettlementService
}

// NewAuctionService 创建竞拍服务
func NewAuctionService(auctionDAO *dao.AuctionDAO) *AuctionService {
	return &AuctionService{
		auctionDAO:        auctionDAO,
		settlementService: NewAuctionSettlementService(auctionDAO, nil),
	}
}

// SetBidDAO 设置出价DAO
func (s *AuctionService) SetBidDAO(bidDAO *dao.BidDAO) {
	s.bidDAO = bidDAO
	s.settlementService.SetBidDAO(bidDAO)
}

// SetNotificationSender 设置通知发送服务
func (s *AuctionService) SetNotificationSender(sender NotificationSender) {
	s.settlementService.SetNotificationSender(sender)
}

func (s *AuctionService) SetOrderCreator(creator AuctionOrderCreator) {
	s.settlementService.SetOrderCreator(creator)
}

func (s *AuctionService) SetSettlementService(settlementService *AuctionSettlementService) {
	s.settlementService = settlementService
	if s.bidDAO != nil {
		s.settlementService.SetBidDAO(s.bidDAO)
	}
}

// SetSkyLampDAO 设置点天灯DAO（用于更新统计数据）
func (s *AuctionService) SetSkyLampDAO(skyLampDAO *dao.SkyLampDAO) {
	// AuctionService暂时不需要SkyLampDAO
	// 保留此方法以备将来扩展
}

// CreateAuctionRequest 创建竞拍请求
type CreateAuctionRequest struct {
	ProductID    int64
	LiveStreamID *int64
	CreatorID    *int64
	StartTime    time.Time
	EndTime      time.Time
}

// CreateAuction 创建竞拍
func (s *AuctionService) CreateAuction(ctx context.Context, req *CreateAuctionRequest) (*model.Auction, error) {
	if req.EndTime.Before(req.StartTime) {
		return nil, errors.New("结束时间不能早于开始时间")
	}

	auction := &model.Auction{
		ProductID:    req.ProductID,
		LiveStreamID: req.LiveStreamID,
		CreatorID:    req.CreatorID,
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

// CancelAuctionByCreator cancels an auction only when it belongs to creatorID.
func (s *AuctionService) CancelAuctionByCreator(ctx context.Context, id, creatorID int64) error {
	auction, err := s.auctionDAO.GetByIDAndCreatorID(ctx, id, creatorID)
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

	return s.auctionDAO.Update(ctx, auction)
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

	return s.auctionDAO.Update(ctx, auction)
}

// EndAuction 结束竞拍
func (s *AuctionService) EndAuction(ctx context.Context, id int64) error {
	if err := s.auctionDAO.DB().Transaction(func(tx *gorm.DB) error {
		txAuctionDAO := s.auctionDAO.WithTx(tx)
		auction, err := txAuctionDAO.GetByID(ctx, id)
		if err != nil {
			return err
		}

		sm := NewStateMachine(auction)
		if err := sm.Transition(model.AuctionStatusEnded); err != nil {
			return err
		}
		if err := s.persistWinnerFromHighestBid(ctx, tx, auction); err != nil {
			return err
		}

		if err := txAuctionDAO.Update(ctx, auction); err != nil {
			return err
		}
		return s.settlementService.CreatePendingTaskWithTx(ctx, tx, id)
	}); err != nil {
		return err
	}

	return s.settlementService.FinalizeEndedAuction(ctx, id)
}

func (s *AuctionService) persistWinnerFromHighestBid(ctx context.Context, tx *gorm.DB, auction *model.Auction) error {
	if auction.WinnerID != nil && *auction.WinnerID > 0 {
		return nil
	}
	if s.bidDAO == nil {
		return nil
	}
	bids, err := s.bidDAO.WithTx(tx).GetRanking(ctx, auction.ID, 1)
	if err != nil {
		return err
	}
	if len(bids) == 0 {
		return nil
	}
	winnerID := bids[0].UserID
	auction.WinnerID = &winnerID
	return nil
}

// CheckAndStartAuctions 检查并开始应该开始的竞拍
func (s *AuctionService) CheckAndStartAuctions(ctx context.Context) error {
	now := auctionBusinessNow()
	auctions, err := s.auctionDAO.GetPendingAuctionsToStart(ctx, now)
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
	now := auctionBusinessNow()
	auctions, err := s.auctionDAO.GetExpiredAuctions(ctx, now)
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

func (s *AuctionService) RetryUnfinishedSettlements(ctx context.Context, limit int) error {
	return s.settlementService.RetryUnfinished(ctx, limit)
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

func (s *AuctionService) ListAdminAuctions(ctx context.Context, status *model.AuctionStatus, page, pageSize int, creatorID *int64) ([]model.Auction, int64, error) {
	return s.auctionDAO.ListAdminScoped(ctx, status, page, pageSize, creatorID)
}

func (s *AuctionService) GetAdminAuction(ctx context.Context, id int64, creatorID *int64) (*model.Auction, error) {
	if creatorID != nil {
		return s.auctionDAO.GetByIDAndCreatorID(ctx, id, *creatorID)
	}
	return s.auctionDAO.GetByID(ctx, id)
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
