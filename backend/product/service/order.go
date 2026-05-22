package service

import (
	"context"
	"errors"

	"product-service/dao"
	"product-service/model"
)

// OrderService 订单服务
type OrderService struct {
	orderDAO     *dao.OrderDAO
	historyDAO   *dao.HistoryDAO
}

// NewOrderService 创建订单服务
func NewOrderService(orderDAO *dao.OrderDAO, historyDAO *dao.HistoryDAO) *OrderService {
	return &OrderService{
		orderDAO:   orderDAO,
		historyDAO: historyDAO,
	}
}

// CreateOrder 创建订单
func (s *OrderService) CreateOrder(ctx context.Context, auctionID, productID, winnerID int64, finalPrice float64) (*model.Order, error) {
	order := &model.Order{
		AuctionID:  auctionID,
		ProductID:  productID,
		WinnerID:   winnerID,
		FinalPrice: finalPrice,
		Status:     model.OrderStatusPending,
	}

	if err := s.orderDAO.Create(ctx, order); err != nil {
		return nil, err
	}

	return order, nil
}

// GetOrder 获取订单详情
func (s *OrderService) GetOrder(ctx context.Context, id int64) (*model.Order, error) {
	return s.orderDAO.GetByID(ctx, id)
}

// ListOrders 获取订单列表
func (s *OrderService) ListOrders(ctx context.Context, userID *int64, page, pageSize int) ([]model.Order, int64, error) {
	return s.orderDAO.List(ctx, userID, page, pageSize)
}

// PayOrder 支付订单（模拟）
func (s *OrderService) PayOrder(ctx context.Context, id int64) (*model.Order, error) {
	order, err := s.orderDAO.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if order.Status != model.OrderStatusPending {
		return nil, errors.New("订单状态不允许支付")
	}

	if err := s.orderDAO.UpdateStatus(ctx, id, model.OrderStatusPaid); err != nil {
		return nil, err
	}

	return s.orderDAO.GetByID(ctx, id)
}

// ShipOrder 发货（模拟）
func (s *OrderService) ShipOrder(ctx context.Context, id int64) (*model.Order, error) {
	order, err := s.orderDAO.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if order.Status != model.OrderStatusPaid {
		return nil, errors.New("订单状态不允许发货")
	}

	if err := s.orderDAO.UpdateStatus(ctx, id, model.OrderStatusShipped); err != nil {
		return nil, err
	}

	return s.orderDAO.GetByID(ctx, id)
}

// GetUserHistory 获取用户竞拍历史
func (s *OrderService) GetUserHistory(ctx context.Context, userID int64, page, pageSize int) ([]dao.UserHistoryItem, int64, error) {
	// 使用 HistoryDAO 查询真实数据
	if s.historyDAO != nil {
		items, total, err := s.historyDAO.QueryUserHistory(ctx, userID, page, pageSize)
		if err != nil {
			// 尝试使用备用方案
			items, total, err = s.historyDAO.QueryUserHistoryGORM(ctx, userID, page, pageSize)
			if err != nil {
				return nil, 0, err
			}
		}

		return items, total, nil
	}

	// 降级：返回空列表
	return []dao.UserHistoryItem{}, 0, nil
}
