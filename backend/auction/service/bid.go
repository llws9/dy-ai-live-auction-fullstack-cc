package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"auction-service/dao"
	"auction-service/lock"
	"auction-service/model"
	"auction-service/pkg/metrics"
	"auction-service/websocket"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// isVersionConflictError 检查是否是乐观锁版本冲突错误
func isVersionConflictError(err error) bool {
	return errors.Is(err, dao.ErrVersionConflict)
}

// BidService 出价服务
type BidService struct {
	auctionDAO         *dao.AuctionDAO
	bidDAO             *dao.BidDAO
	ruleDAO            *dao.AuctionRuleDAO
	userDAO            *dao.UserDAO
	hub                *websocket.Hub
	rankThrottle       *RankingThrottle
	notificationSender NotificationSender      // 通知发送接口
	metrics            *metrics.AuctionMetrics // 新增：指标收集器
	skyLampTrigger     SkyLampTrigger          // 点天灯触发接口
	settlementService  *AuctionSettlementService
}

// SkyLampTrigger 点天灯触发接口
type SkyLampTrigger interface {
	TriggerAutoBid(ctx context.Context, auctionID int64, currentPrice float64, increment float64) error
}

// NewBidService 创建出价服务
func NewBidService(auctionDAO *dao.AuctionDAO, bidDAO *dao.BidDAO, ruleDAO *dao.AuctionRuleDAO, userDAO *dao.UserDAO) *BidService {
	return &BidService{
		auctionDAO:        auctionDAO,
		bidDAO:            bidDAO,
		ruleDAO:           ruleDAO,
		userDAO:           userDAO,
		rankThrottle:      NewRankingThrottle(),
		settlementService: NewAuctionSettlementService(auctionDAO, bidDAO),
	}
}

// SetMetrics 设置指标收集器
func (s *BidService) SetMetrics(m *metrics.AuctionMetrics) {
	s.metrics = m
}

// SetHub 设置 WebSocket Hub
func (s *BidService) SetHub(hub *websocket.Hub) {
	s.hub = hub
}

// SetNotificationSender 设置通知发送服务
func (s *BidService) SetNotificationSender(sender NotificationSender) {
	s.notificationSender = sender
	if s.settlementService != nil {
		s.settlementService.SetNotificationSender(sender)
	}
}

// SetSkyLampTrigger 设置点天灯触发服务
func (s *BidService) SetSkyLampTrigger(trigger SkyLampTrigger) {
	s.skyLampTrigger = trigger
}

func (s *BidService) SetSettlementService(settlementService *AuctionSettlementService) {
	s.settlementService = settlementService
}

// PlaceBidRequest 出价请求
type PlaceBidRequest struct {
	AuctionID          int64
	UserID             int64
	Amount             decimal.Decimal
	SkipSkyLampTrigger bool // 点天灯自动跟价调用时设为true，避免递归
}

// PlaceBidResult 出价结果
type PlaceBidResult struct {
	Success      bool            `json:"success"`
	Message      string          `json:"message"`
	CurrentPrice decimal.Decimal `json:"current_price"`
	Rank         int             `json:"rank"`
	WinnerID     int64           `json:"winner_id"`
}

// PlaceBid 出价
func (s *BidService) PlaceBid(ctx context.Context, req *PlaceBidRequest) (*PlaceBidResult, error) {
	start := time.Now()

	// 记录并发出价计数（新增）
	if s.metrics != nil {
		s.metrics.IncConcurrentBids()
		defer s.metrics.DecConcurrentBids() // 确保函数结束时总是递减
	}

	// 1. 校验用户是否存在（逻辑外键校验）
	if s.userDAO != nil {
		exists, err := s.userDAO.Exists(ctx, req.UserID)
		if err != nil {
			return nil, fmt.Errorf("校验用户失败: %w", err)
		}
		if !exists {
			return &PlaceBidResult{
				Success: false,
				Message: fmt.Sprintf("用户 %d 不存在，请先创建用户", req.UserID),
			}, nil
		}
	}

	// 2. 获取竞拍信息
	auction, err := s.auctionDAO.GetByID(ctx, req.AuctionID)
	if err != nil {
		return nil, fmt.Errorf("竞拍不存在: %w", err)
	}

	// 3. 检查竞拍状态
	sm := NewStateMachine(auction)
	if !sm.CanBid() {
		return &PlaceBidResult{
			Success: false,
			Message: "竞拍已结束或已取消，无法出价",
		}, nil
	}

	// 4. 获取竞拍规则
	rule, err := s.ruleDAO.GetByProductID(ctx, auction.ProductID)
	if err != nil {
		return nil, fmt.Errorf("获取竞拍规则失败: %w", err)
	}
	if rule == nil {
		return nil, errors.New("竞拍规则不存在")
	}

	// 5. 校验出价金额
	minBidAmount := auction.CurrentPrice.Add(rule.Increment)
	if req.Amount.LessThan(minBidAmount) {
		return &PlaceBidResult{
			Success: false,
			Message: fmt.Sprintf("出价金额不足，最低出价为 %s 元", minBidAmount.StringFixed(2)),
		}, nil
	}

	// 6. 检查封顶价
	if rule.CapPrice != nil && rule.CapPrice.GreaterThan(decimal.Zero) && req.Amount.GreaterThanOrEqual(*rule.CapPrice) {
		// 达到封顶价，需要获取分布式锁保护
		bidLock := lock.NewAuctionBidLock(dao.GetRedis(), req.AuctionID)
		if err := bidLock.Acquire(ctx); err != nil {
			// 区分锁占用（业务正常）和 Redis 错误（基础设施异常）
			if errors.Is(err, lock.ErrLockNotAcquired) {
				return &PlaceBidResult{
					Success: false,
					Message: "出价过于频繁，请稍后再试",
				}, nil
			}
			return nil, fmt.Errorf("获取出价锁失败: %w", err)
		}
		defer bidLock.Release(ctx)

		// 锁后重新获取 auction 以使用正确的版本号
		auction, err = s.auctionDAO.GetByID(ctx, req.AuctionID)
		if err != nil {
			return nil, fmt.Errorf("重新获取竞拍失败: %w", err)
		}

		// 双重检查：确认竞拍状态仍然可以出价
		sm := NewStateMachine(auction)
		if !sm.CanBid() {
			return &PlaceBidResult{
				Success: false,
				Message: "竞拍已结束，无法出价",
			}, nil
		}

		// 直接成交
		return s.handleCapPriceBid(ctx, auction, req, *rule.CapPrice)
	}

	// 7. 获取分布式锁
	bidLock := lock.NewAuctionBidLock(dao.GetRedis(), req.AuctionID)
	if err := bidLock.Acquire(ctx); err != nil {
		// 区分锁占用（业务正常）和 Redis 错误（基础设施异常）
		if errors.Is(err, lock.ErrLockNotAcquired) {
			return &PlaceBidResult{
				Success: false,
				Message: "出价过于频繁，请稍后再试",
			}, nil
		}
		return nil, fmt.Errorf("获取出价锁失败: %w", err)
	}
	defer bidLock.Release(ctx)

	// 8. 重新获取竞拍信息（防止并发问题）
	auction, err = s.auctionDAO.GetByID(ctx, req.AuctionID)
	if err != nil {
		return nil, err
	}

	// 9. 再次校验（双重检查）
	minBidAmount = auction.CurrentPrice.Add(rule.Increment)
	if req.Amount.LessThan(minBidAmount) {
		return &PlaceBidResult{
			Success: false,
			Message: fmt.Sprintf("已被超越，当前最低出价为 %s 元", minBidAmount.StringFixed(2)),
		}, nil
	}

	// 10. 创建出价记录 + 更新竞拍价格（同一事务，乐观锁带重试）
	// 保存前中标者信息，用于发送出价超越通知
	previousWinnerID := auction.WinnerID
	previousPrice := auction.CurrentPrice

	// 乐观锁重试配置
	const maxRetries = 3
	var lastErr error
	for retry := 0; retry < maxRetries; retry++ {
		// 如果是重试，重新获取 auction 获取最新版本号
		if retry > 0 {
			auction, err = s.auctionDAO.GetByID(ctx, req.AuctionID)
			if err != nil {
				lastErr = fmt.Errorf("重试获取竞拍失败: %w", err)
				continue
			}
			// 重试时再次校验最低出价
			minBidAmount = auction.CurrentPrice.Add(rule.Increment)
			if req.Amount.LessThan(minBidAmount) {
				return &PlaceBidResult{
					Success: false,
					Message: fmt.Sprintf("已被超越，当前最低出价为 %s 元", minBidAmount.StringFixed(2)),
				}, nil
			}
		}

		expectedVersion := auction.Version
		txErr := s.auctionDAO.DB().Transaction(func(tx *gorm.DB) error {
			// 在事务中创建出价记录
			bid := &model.Bid{
				AuctionID: req.AuctionID,
				UserID:    req.UserID,
				Amount:    req.Amount,
			}
			if err := s.bidDAO.WithTx(tx).Create(ctx, bid); err != nil {
				return fmt.Errorf("创建出价记录失败: %w", err)
			}

			// 在事务中更新竞拍价格（乐观锁）
			if err := s.auctionDAO.WithTx(tx).UpdatePrice(ctx, req.AuctionID, req.Amount, req.UserID, expectedVersion); err != nil {
				return err
			}
			return nil
		})

		if txErr == nil {
			lastErr = nil
			break
		}

		lastErr = txErr
		// 检查是否是版本冲突错误（可重试）
		if isVersionConflictError(txErr) {
			continue
		}
		// 其他错误直接返回
		return nil, fmt.Errorf("出价失败: %w", txErr)
	}

	if lastErr != nil {
		return nil, fmt.Errorf("出价失败（重试%d次后）: %w", maxRetries, lastErr)
	}

	// 11.1 发送出价超越通知
	if s.notificationSender != nil && previousWinnerID != nil && *previousWinnerID > 0 && *previousWinnerID != req.UserID {
		notifyUserID := *previousWinnerID
		go func() {
			// 异步发送通知，不阻塞主流程；Immediately=true 确保 WebSocket 立即推送
			_ = s.notificationSender.SendNotification(context.Background(), &model.NotificationRequest{
				UserID:      notifyUserID,
				Type:        model.NotificationTypeBidOutbid,
				Title:       "出价被超越",
				Content:     fmt.Sprintf("您在竞拍中的出价 %s 元已被超越，当前最高价为 %s 元", previousPrice.StringFixed(2), req.Amount.StringFixed(2)),
				Immediately: true,
				Data: map[string]interface{}{
					"auction_id": req.AuctionID,
					"old_bid":    previousPrice.StringFixed(2),
					"new_bid":    req.Amount.StringFixed(2),
				},
			})
		}()
	}

	// 11.2 触发点天灯自动跟价（非点天灯自动跟价调用才触发，避免递归）
	if s.skyLampTrigger != nil && !req.SkipSkyLampTrigger {
		go func() {
			// 异步触发，不阻塞主流程
			_ = s.skyLampTrigger.TriggerAutoBid(context.Background(), req.AuctionID, decimalToFloat(req.Amount), decimalToFloat(rule.Increment))
		}()
	}

	// 11.5 异步检查并触发延时
	// 改为 goroutine 后台执行,避免在分布式锁持有期内做 DB 查询+事务,
	// 缩短锁占用时间;事务的乐观锁/状态机保证多次触发不会重复延时。
	if rule.TriggerDelayBefore > 0 {
		auctionID := req.AuctionID
		triggerDelayBefore := rule.TriggerDelayBefore
		maxDelayTime := rule.MaxDelayTime
		delayDuration := rule.DelayDuration
		productID := auction.ProductID
		go s.tryExtendAuction(auctionID, productID, triggerDelayBefore, maxDelayTime, delayDuration)
	}

	// 12. 获取用户排名
	rank, err := s.getUserRank(ctx, req.AuctionID, req.UserID)
	if err != nil {
		rank = 1 // 默认第一名
	}

	// 13. 广播排名更新
	s.broadcastRanking(ctx, req.AuctionID)

	// 14. 返回成功结果
	// 记录出价成功指标（新增）
	if s.metrics != nil {
		s.metrics.RecordBidLatency(req.AuctionID, start, true)
		s.metrics.RecordBid(req.AuctionID, decimalToFloat(req.Amount), true)
		s.metrics.RecordBidUser(req.AuctionID)
	}

	return &PlaceBidResult{
		Success:      true,
		Message:      "出价成功",
		CurrentPrice: req.Amount,
		Rank:         rank,
		WinnerID:     req.UserID,
	}, nil
}

// tryExtendAuction 后台异步检测并触发延时。
// 设计要点:
//   - 使用独立 context (默认 5s 超时),解耦于出价请求的生命周期;
//   - 重新读取最新 auction/rule,避免基于陈旧数据决策;
//   - 通过状态机 ShouldTriggerDelay/CanDelay 防止重复延时,事务保证 ExtendEndTime+UpdateStatus 原子执行。
func (s *BidService) tryExtendAuction(auctionID, productID int64, triggerDelayBefore, maxDelayTime, delayDuration int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	auction, err := s.auctionDAO.GetByID(ctx, auctionID)
	if err != nil {
		return
	}

	// 重新获取 rule,避免使用陈旧规则数据;失败则回退到调用方传入的快照参数
	if freshRule, ruleErr := s.ruleDAO.GetByProductID(ctx, productID); ruleErr == nil {
		triggerDelayBefore = freshRule.TriggerDelayBefore
		maxDelayTime = freshRule.MaxDelayTime
		delayDuration = freshRule.DelayDuration
	}

	sm := NewStateMachine(auction)
	if !sm.ShouldTriggerDelay(triggerDelayBefore) || !sm.CanDelay(maxDelayTime) {
		return
	}

	// 计算可延时时长,确保不超过 MaxDelayTime 限制
	remainingDelay := sm.GetRemainingDelayTime(maxDelayTime)
	actualDelay := delayDuration
	if actualDelay > remainingDelay {
		actualDelay = remainingDelay
	}
	if actualDelay <= 0 {
		return
	}

	txErr := s.auctionDAO.DB().Transaction(func(tx *gorm.DB) error {
		txDAO := s.auctionDAO.WithTx(tx)
		if err := txDAO.ExtendEndTime(ctx, auctionID, actualDelay); err != nil {
			return err
		}
		if auction.Status == model.AuctionStatusOngoing {
			if err := txDAO.UpdateStatus(ctx, auctionID, model.AuctionStatusDelayed); err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		return
	}

	fmt.Printf("Auction %d delayed by %d seconds\n", auctionID, actualDelay)

	updated, err := s.auctionDAO.GetByID(ctx, auctionID)
	if err == nil {
		remainingDelay := maxDelayTime - updated.DelayUsed
		if remainingDelay < 0 {
			remainingDelay = 0
		}
		s.broadcastDelayTriggered(auctionID, actualDelay, updated.EndTime, remainingDelay, maxDelayTime)
	}

	if s.metrics != nil {
		s.metrics.RecordDelayTriggered(auctionID)
	}
}

// handleCapPriceBid 处理封顶价出价（使用事务保证原子性）
func (s *BidService) handleCapPriceBid(ctx context.Context, auction *model.Auction, req *PlaceBidRequest, capPrice decimal.Decimal) (*PlaceBidResult, error) {
	// 使用事务确保价格更新和状态更新的原子性
	var result *PlaceBidResult
	var txErr error

	txErr = s.auctionDAO.DB().Transaction(func(tx *gorm.DB) error {
		// 在事务中创建出价记录
		bid := &model.Bid{
			AuctionID: req.AuctionID,
			UserID:    req.UserID,
			Amount:    capPrice,
		}

		if err := s.bidDAO.WithTx(tx).Create(ctx, bid); err != nil {
			return fmt.Errorf("创建出价记录失败: %w", err)
		}

		// 在事务中更新竞拍价格和状态
		txDAO := s.auctionDAO.WithTx(tx)

		// 更新竞拍价格（使用乐观锁）
		if err := txDAO.UpdatePrice(ctx, req.AuctionID, capPrice, req.UserID, auction.Version); err != nil {
			return fmt.Errorf("更新竞拍价格失败: %w", err)
		}

		// 更新竞拍状态为已结束
		if err := txDAO.UpdateStatus(ctx, req.AuctionID, model.AuctionStatusEnded); err != nil {
			return fmt.Errorf("更新竞拍状态失败: %w", err)
		}

		if s.settlementService == nil {
			return errors.New("竞拍收尾服务未初始化")
		}
		if err := s.settlementService.CreatePendingTaskWithTx(ctx, tx, req.AuctionID); err != nil {
			return fmt.Errorf("创建竞拍收尾任务失败: %w", err)
		}

		result = &PlaceBidResult{
			Success:      true,
			Message:      "达到封顶价，竞拍成交！",
			CurrentPrice: capPrice,
			Rank:         1,
			WinnerID:     req.UserID,
		}
		return nil
	})

	if txErr != nil {
		return nil, txErr
	}

	if err := s.settlementService.FinalizeEndedAuction(ctx, req.AuctionID); err != nil {
		return nil, err
	}

	return result, nil
}

func decimalToFloat(v decimal.Decimal) float64 {
	f, _ := v.Float64()
	return f
}

// getUserRank 获取用户排名
func (s *BidService) getUserRank(ctx context.Context, auctionID, userID int64) (int, error) {
	bids, err := s.bidDAO.GetRanking(ctx, auctionID, 100)
	if err != nil {
		return 0, err
	}

	for i, bid := range bids {
		if bid.UserID == userID {
			return i + 1, nil
		}
	}

	return 0, nil
}

// GetRanking 获取出价排名
func (s *BidService) GetRanking(ctx context.Context, auctionID int64, limit int) ([]model.Bid, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	return s.bidDAO.GetRanking(ctx, auctionID, limit)
}

// broadcastRanking 广播排名更新
func (s *BidService) broadcastRanking(ctx context.Context, auctionID int64) {
	if s.hub == nil {
		return
	}

	// 节流检查：每 200ms 最多推送一次
	if !s.rankThrottle.ShouldSend(auctionID) {
		return
	}

	// 获取排名列表
	bids, err := s.bidDAO.GetRanking(ctx, auctionID, 10)
	if err != nil {
		return
	}

	// 转换为排名项
	rankItems := make([]websocket.RankItem, len(bids))

	// 批量查询用户名，避免 N+1 查询
	var userMap map[int64]*model.User
	if s.userDAO != nil && len(bids) > 0 {
		userIDs := make([]int64, 0, len(bids))
		for _, bid := range bids {
			userIDs = append(userIDs, bid.UserID)
		}
		userMap, _ = s.userDAO.GetByIDs(ctx, userIDs)
	}

	for i, bid := range bids {
		rankItems[i] = websocket.RankItem{
			Rank:   i + 1,
			UserID: bid.UserID,
			Amount: bid.Amount,
		}
		if user, ok := userMap[bid.UserID]; ok && user != nil {
			rankItems[i].UserName = user.Name
		}
	}

	// 广播排名更新消息
	message := websocket.NewRankUpdateMessage(auctionID, rankItems)
	s.hub.BroadcastToRoom(auctionID, message)
}

// broadcastDelayTriggered 广播防狙击延时消息，使前端实时更新倒计时。
// 仅依赖 hub，便于单测；hub 为 nil 时安全跳过。
func (s *BidService) broadcastDelayTriggered(auctionID int64, delayDuration int, newEndTime time.Time, remainingDelay, maxDelay int) {
	if s.hub == nil {
		return
	}
	msg := websocket.NewDelayTriggeredMessage(&websocket.DelayTriggeredData{
		AuctionID:      auctionID,
		DelayDuration:  delayDuration,
		NewEndTime:     newEndTime.UnixMilli(),
		RemainingDelay: remainingDelay,
		MaxDelay:       maxDelay,
	})
	s.hub.BroadcastToRoom(auctionID, msg)
}
