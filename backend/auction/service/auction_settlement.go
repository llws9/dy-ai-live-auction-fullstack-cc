package service

import (
	"context"
	"errors"
	"fmt"

	"auction-service/dao"
	"auction-service/model"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type AuctionOrderCreator interface {
	CreateOrderFromAuctionResult(ctx context.Context, req model.AuctionOrderRequest) error
}

type AuctionSettlementService struct {
	auctionDAO         *dao.AuctionDAO
	bidDAO             *dao.BidDAO
	taskDAO            *dao.AuctionSettlementTaskDAO
	notificationSender NotificationSender
	orderCreator       AuctionOrderCreator
}

func NewAuctionSettlementService(auctionDAO *dao.AuctionDAO, bidDAO *dao.BidDAO) *AuctionSettlementService {
	return &AuctionSettlementService{
		auctionDAO: auctionDAO,
		bidDAO:     bidDAO,
		taskDAO:    dao.NewAuctionSettlementTaskDAO(auctionDAO.DB()),
	}
}

func (s *AuctionSettlementService) SetBidDAO(bidDAO *dao.BidDAO) {
	s.bidDAO = bidDAO
}

func (s *AuctionSettlementService) SetNotificationSender(sender NotificationSender) {
	s.notificationSender = sender
}

func (s *AuctionSettlementService) SetOrderCreator(creator AuctionOrderCreator) {
	s.orderCreator = creator
}

func (s *AuctionSettlementService) CreatePendingTaskWithTx(ctx context.Context, tx *gorm.DB, auctionID int64) error {
	return s.taskDAO.WithTx(tx).CreatePendingIfNotExists(ctx, auctionID)
}

func (s *AuctionSettlementService) FinalizeEndedAuction(ctx context.Context, auctionID int64) error {
	task, err := s.taskDAO.EnsurePending(ctx, auctionID)
	if err != nil {
		return err
	}
	if task.Status == model.AuctionSettlementTaskStatusDone {
		return nil
	}

	auction, err := s.auctionDAO.GetByID(ctx, auctionID)
	if err != nil {
		return err
	}

	if task.Status == model.AuctionSettlementTaskStatusPending {
		hasResult, err := s.createOrderForAuctionResult(ctx, auction)
		if err != nil {
			return err
		}
		if !hasResult {
			return s.taskDAO.UpdateStatus(ctx, auctionID, model.AuctionSettlementTaskStatusDone)
		}
		if err := s.taskDAO.UpdateStatus(ctx, auctionID, model.AuctionSettlementTaskStatusOrderDone); err != nil {
			return err
		}
	}

	if err := s.SendAuctionResultNotifications(ctx, auction); err != nil {
		return err
	}
	return s.taskDAO.UpdateStatus(ctx, auctionID, model.AuctionSettlementTaskStatusDone)
}

func (s *AuctionSettlementService) RetryUnfinished(ctx context.Context, limit int) error {
	tasks, err := s.taskDAO.ListUnfinished(ctx, limit)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if err := s.FinalizeEndedAuction(ctx, task.AuctionID); err != nil {
			continue
		}
	}
	return nil
}

func (s *AuctionSettlementService) CreateOrderForAuctionResult(ctx context.Context, auction *model.Auction) error {
	_, err := s.createOrderForAuctionResult(ctx, auction)
	return err
}

func (s *AuctionSettlementService) createOrderForAuctionResult(ctx context.Context, auction *model.Auction) (bool, error) {
	winnerID, finalPrice, _, ok, err := s.auctionWinnerResult(ctx, auction)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	if s.orderCreator == nil {
		return true, errors.New("订单创建器未初始化")
	}

	if err := s.orderCreator.CreateOrderFromAuctionResult(ctx, model.AuctionOrderRequest{
		AuctionID:  auction.ID,
		ProductID:  auction.ProductID,
		WinnerID:   winnerID,
		FinalPrice: finalPrice,
	}); err != nil {
		return true, fmt.Errorf("创建中标订单失败: %w", err)
	}
	return true, nil
}

func (s *AuctionSettlementService) SendAuctionResultNotifications(ctx context.Context, auction *model.Auction) error {
	if s.notificationSender == nil {
		return nil
	}
	winnerID, finalPrice, bids, ok, err := s.auctionWinnerResult(ctx, auction)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

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

	var loserRequests []*model.NotificationRequest
	for _, bid := range bids {
		if bid.UserID == winnerID {
			continue
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

	if len(loserRequests) > 0 {
		go func() {
			_ = s.notificationSender.SendBatchNotifications(ctx, loserRequests)
		}()
	}

	return nil
}

func (s *AuctionSettlementService) auctionWinnerResult(ctx context.Context, auction *model.Auction) (winnerID int64, finalPrice decimal.Decimal, bids []model.Bid, ok bool, err error) {
	if s.bidDAO == nil {
		return 0, decimal.Zero, nil, false, nil
	}
	bids, err = s.bidDAO.GetRanking(ctx, auction.ID, 1000)
	if err != nil {
		return 0, decimal.Zero, nil, false, err
	}
	if len(bids) == 0 {
		return 0, decimal.Zero, nil, false, nil
	}
	if auction.WinnerID != nil && *auction.WinnerID > 0 {
		winnerID = *auction.WinnerID
	} else {
		winnerID = bids[0].UserID
	}
	return winnerID, auction.CurrentPrice, bids, true, nil
}
