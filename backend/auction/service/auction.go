package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"auction-service/dao"
	"auction-service/model"
	"auction-service/websocket"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var (
	ErrProductOwnershipMismatch       = errors.New("商品不存在或不属于当前商家")
	ErrProductNotSchedulable          = errors.New("商品未进入竞拍池")
	ErrAuctionRuleNotBound            = errors.New("规则模板不存在或不属于当前商家")
	ErrActiveAuctionExists            = errors.New("该商品已有待开始或进行中的竞拍场次")
	ErrActiveLiveStreamAuctionExists  = errors.New("当前直播间已有待开始或进行中的竞拍场次")
	ErrSoldProductCannotBeReauctioned = errors.New("已成交商品不能再次创建竞拍")
)

const productStatusPublished = 1 // mirrors product-service model.ProductStatusPublished

// AuctionService 竞拍服务
type AuctionService struct {
	auctionDAO        *dao.AuctionDAO
	bidDAO            *dao.BidDAO
	settlementService *AuctionSettlementService
	stateManager      *websocket.StateManager
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
	ProductID      int64
	CreatorID      *int64
	Duration       int
	ProductOwnerID int64
	ProductStatus  int
	RuleBound      bool
	LiveStreamID   int64
	// Deprecated: compatibility for pre-T6 handler/tests. Ignored by CreateAuction;
	// Task 6 will migrate callers to Duration-based scheduling.
	StartTime time.Time
	// Deprecated: compatibility for pre-T6 handler/tests. Ignored by CreateAuction;
	// Task 6 will migrate callers to Duration-based scheduling.
	EndTime time.Time
}

// CreateAuction 创建竞拍
func (s *AuctionService) CreateAuction(ctx context.Context, req *CreateAuctionRequest) (*model.Auction, error) {
	if req == nil {
		return nil, errors.New("创建竞拍请求不能为空")
	}
	if req.CreatorID == nil || *req.CreatorID <= 0 {
		return nil, errors.New("创建者ID非法")
	}
	if req.ProductID <= 0 {
		return nil, errors.New("商品ID非法")
	}
	if req.Duration <= 0 {
		return nil, errors.New("竞拍时长必须大于0")
	}
	if req.ProductOwnerID != *req.CreatorID {
		return nil, ErrProductOwnershipMismatch
	}
	if req.ProductStatus != productStatusPublished {
		return nil, ErrProductNotSchedulable
	}
	if !req.RuleBound {
		return nil, ErrAuctionRuleNotBound
	}
	if req.LiveStreamID <= 0 {
		return nil, errors.New("直播间不可用")
	}

	active, err := s.auctionDAO.GetActiveByProductID(ctx, req.ProductID)
	if err != nil {
		return nil, err
	}
	if active != nil {
		return nil, ErrActiveAuctionExists
	}
	activeInLiveStream, err := s.auctionDAO.GetActiveByLiveStreamID(ctx, req.LiveStreamID)
	if err != nil {
		return nil, err
	}
	if activeInLiveStream != nil {
		return nil, ErrActiveLiveStreamAuctionExists
	}

	latest, err := s.auctionDAO.GetLatestTerminalByProductID(ctx, req.ProductID)
	if err != nil {
		return nil, err
	}
	if latest != nil && latest.Status == model.AuctionStatusEnded && latest.WinnerID != nil {
		return nil, ErrSoldProductCannotBeReauctioned
	}

	now := time.Now()
	liveStreamID := req.LiveStreamID
	auction := &model.Auction{
		ProductID:    req.ProductID,
		LiveStreamID: &liveStreamID,
		CreatorID:    req.CreatorID,
		Status:       model.AuctionStatusPending,
		CurrentPrice: decimal.Zero,
		StartTime:    now,
		EndTime:      now.Add(time.Duration(req.Duration) * time.Second),
		DelayUsed:    0,
	}

	if err := s.auctionDAO.Create(ctx, auction); err != nil {
		if isActiveLiveStreamUniqueConflict(err) {
			return nil, ErrActiveLiveStreamAuctionExists
		}
		if isActiveAuctionUniqueConflict(err) {
			return nil, ErrActiveAuctionExists
		}
		return nil, err
	}

	return auction, nil
}

func isActiveAuctionUniqueConflict(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "uk_active_product")
}

func isActiveLiveStreamUniqueConflict(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "uk_active_live_stream")
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

	if auction, err := s.auctionDAO.GetByID(ctx, id); err == nil {
		s.saveSyncState(ctx, auction)
	}
	return s.settlementService.FinalizeEndedAuction(ctx, id)
}

func (s *AuctionService) saveSyncState(ctx context.Context, auction *model.Auction) {
	_ = SaveAuctionSyncState(ctx, s.stateManager, auction)
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
