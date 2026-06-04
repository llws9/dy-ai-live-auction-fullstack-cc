package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"product-service/dao"
	"product-service/model"
	"product-service/pkg/logger"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// NotificationCallback 通知回调接口（用于Mock触发通知）
type NotificationCallback interface {
	// OnOrderPaid 订单已支付通知
	OnOrderPaid(ctx context.Context, userID int64, orderID int64) error
	// OnOrderShipped 订单已发货通知
	OnOrderShipped(ctx context.Context, userID int64, orderID int64) error
	// OnOrderCompleted 订单已完成通知
	OnOrderCompleted(ctx context.Context, userID int64, orderID int64) error
}

// OrderService 订单服务
type OrderService struct {
	orderDAO             *dao.OrderDAO
	historyDAO           *dao.HistoryDAO
	adminDAO             *dao.OrderAdminDAO
	productDAO           *dao.ProductDAO
	notificationCallback NotificationCallback // 通知回调（Mock触发）
	logger               *logger.Logger
}

// NewOrderService 创建订单服务
func NewOrderService(orderDAO *dao.OrderDAO, historyDAO *dao.HistoryDAO) *OrderService {
	return &OrderService{
		orderDAO:   orderDAO,
		historyDAO: historyDAO,
		logger:     logger.NewLogger("product-service"),
	}
}

// SetAdminOrderDAO 注入 admin 视图 DAO。可选：未注入时 admin 接口返回空列表。
func (s *OrderService) SetAdminOrderDAO(adminDAO *dao.OrderAdminDAO) {
	s.adminDAO = adminDAO
}

// SetProductDAO 注入商品 DAO，用于在订单创建时固化 seller_id。
func (s *OrderService) SetProductDAO(productDAO *dao.ProductDAO) {
	s.productDAO = productDAO
}

// SetNotificationCallback 设置通知回调
func (s *OrderService) SetNotificationCallback(callback NotificationCallback) {
	s.notificationCallback = callback
}

// CreateOrder 创建订单
func (s *OrderService) CreateOrder(ctx context.Context, auctionID, productID, winnerID int64, finalPrice decimal.Decimal) (*model.Order, error) {
	var sellerID *int64
	if s.productDAO != nil {
		product, err := s.productDAO.GetByID(ctx, productID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("商品不存在: %d", productID)
			}
			s.logger.LogOperation(ctx, logger.OperationCreate, logger.ObjectOrder, fmt.Sprintf("%d", productID), false, err)
			return nil, err
		}
		if product.OwnerID == nil {
			err := fmt.Errorf("商品缺少商家归属: %d", productID)
			s.logger.LogOperation(ctx, logger.OperationCreate, logger.ObjectOrder, fmt.Sprintf("%d", productID), false, err)
			return nil, err
		}
		sellerID = product.OwnerID
	}
	order := &model.Order{
		AuctionID:  auctionID,
		ProductID:  productID,
		SellerID:   sellerID,
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

func (s *OrderService) GetOrderForUser(ctx context.Context, id, userID int64) (*model.Order, error) {
	return s.orderDAO.GetByIDAndWinnerID(ctx, id, userID)
}

// ListOrders 获取订单列表
func (s *OrderService) ListOrders(ctx context.Context, userID *int64, page, pageSize int) ([]model.Order, int64, error) {
	if s.orderDAO == nil {
		return []model.Order{}, 0, nil
	}
	return s.orderDAO.List(ctx, userID, page, pageSize)
}

// GetSummary 获取用户订单触点汇总
func (s *OrderService) GetSummary(ctx context.Context, userID int64) (*model.OrderSummaryResponse, error) {
	pending, err := s.orderDAO.CountByWinnerAndStatus(ctx, userID, model.OrderStatusPending)
	if err != nil {
		return nil, err
	}
	return &model.OrderSummaryResponse{
		PendingPayment: pending,
		WonNotPaid:     pending,
	}, nil
}

// PayOrder 支付订单（模拟）
func (s *OrderService) PayOrder(ctx context.Context, id int64) (*model.Order, error) {
	start := time.Now()

	order, err := s.orderDAO.GetByID(ctx, id)
	if err != nil {
		s.logger.LogOperation(ctx, logger.OperationPay, logger.ObjectOrder, fmt.Sprintf("%d", id), false, err)
		return nil, err
	}

	if order.Status != model.OrderStatusPending {
		err := errors.New("订单状态不允许支付")
		s.logger.LogOperationWithData(ctx, logger.OperationPay, logger.ObjectOrder, fmt.Sprintf("%d", id),
			false, err, map[string]interface{}{
				"current_status": order.Status,
			}, nil)
		return nil, err
	}

	if err := s.orderDAO.UpdateStatus(ctx, id, model.OrderStatusPaid); err != nil {
		s.logger.LogOperation(ctx, logger.OperationPay, logger.ObjectOrder, fmt.Sprintf("%d", id), false, err)
		return nil, err
	}

	// Mock触发：发送订单已支付通知
	if s.notificationCallback != nil {
		go func() {
			_ = s.notificationCallback.OnOrderPaid(ctx, order.WinnerID, id)
		}()
	}

	s.logger.LogOperationWithData(ctx, logger.OperationPay, logger.ObjectOrder, fmt.Sprintf("%d", id),
		true, nil, map[string]interface{}{
			"order_id":    id,
			"auction_id":  order.AuctionID,
			"product_id":  order.ProductID,
			"winner_id":   order.WinnerID,
			"final_price": order.FinalPrice,
			"duration_ms": time.Since(start).Milliseconds(),
		}, nil)

	return s.orderDAO.GetByID(ctx, id)
}

func (s *OrderService) PayOrderForUser(ctx context.Context, id, userID int64) (*model.Order, error) {
	order, err := s.GetOrderForUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if order.Status != model.OrderStatusPending {
		return nil, errors.New("订单状态不允许支付")
	}
	if err := s.orderDAO.UpdateStatus(ctx, id, model.OrderStatusPaid); err != nil {
		return nil, err
	}
	if s.notificationCallback != nil {
		go func() {
			_ = s.notificationCallback.OnOrderPaid(ctx, order.WinnerID, id)
		}()
	}
	return s.GetOrderForUser(ctx, id, userID)
}

// ShipOrder 发货（模拟）
func (s *OrderService) ShipOrder(ctx context.Context, id int64) (*model.Order, error) {
	start := time.Now()

	order, err := s.orderDAO.GetByID(ctx, id)
	if err != nil {
		s.logger.LogOperation(ctx, logger.OperationShip, logger.ObjectOrder, fmt.Sprintf("%d", id), false, err)
		return nil, err
	}

	if order.Status != model.OrderStatusPaid {
		err := errors.New("订单状态不允许发货")
		s.logger.LogOperationWithData(ctx, logger.OperationShip, logger.ObjectOrder, fmt.Sprintf("%d", id),
			false, err, map[string]interface{}{
				"current_status": order.Status,
			}, nil)
		return nil, err
	}

	if err := s.orderDAO.UpdateStatus(ctx, id, model.OrderStatusShipped); err != nil {
		s.logger.LogOperation(ctx, logger.OperationShip, logger.ObjectOrder, fmt.Sprintf("%d", id), false, err)
		return nil, err
	}

	// Mock触发：发送订单已发货通知
	if s.notificationCallback != nil {
		go func() {
			_ = s.notificationCallback.OnOrderShipped(ctx, order.WinnerID, id)
		}()
	}

	s.logger.LogOperationWithData(ctx, logger.OperationShip, logger.ObjectOrder, fmt.Sprintf("%d", id),
		true, nil, map[string]interface{}{
			"order_id":    id,
			"auction_id":  order.AuctionID,
			"product_id":  order.ProductID,
			"winner_id":   order.WinnerID,
			"duration_ms": time.Since(start).Milliseconds(),
		}, nil)

	return s.orderDAO.GetByID(ctx, id)
}

func (s *OrderService) ShipOrderForSeller(ctx context.Context, id, sellerID int64) (*model.Order, error) {
	order, err := s.orderDAO.GetByIDAndSellerID(ctx, id, sellerID)
	if err != nil {
		return nil, err
	}
	if order.Status != model.OrderStatusPaid {
		return nil, errors.New("订单状态不允许发货")
	}
	if err := s.orderDAO.ShipOrderForSeller(ctx, id, sellerID); err != nil {
		return nil, err
	}
	if s.notificationCallback != nil {
		go func() {
			_ = s.notificationCallback.OnOrderShipped(ctx, order.WinnerID, id)
		}()
	}
	return s.orderDAO.GetByIDAndSellerID(ctx, id, sellerID)
}

// CompleteOrder 完成订单（模拟）
func (s *OrderService) CompleteOrder(ctx context.Context, id int64) (*model.Order, error) {
	start := time.Now()

	order, err := s.orderDAO.GetByID(ctx, id)
	if err != nil {
		s.logger.LogOperation(ctx, logger.OperationComplete, logger.ObjectOrder, fmt.Sprintf("%d", id), false, err)
		return nil, err
	}

	if order.Status != model.OrderStatusShipped {
		err := errors.New("订单状态不允许完成")
		s.logger.LogOperationWithData(ctx, logger.OperationComplete, logger.ObjectOrder, fmt.Sprintf("%d", id),
			false, err, map[string]interface{}{
				"current_status": order.Status,
			}, nil)
		return nil, err
	}

	if err := s.orderDAO.UpdateStatus(ctx, id, model.OrderStatusCompleted); err != nil {
		s.logger.LogOperation(ctx, logger.OperationComplete, logger.ObjectOrder, fmt.Sprintf("%d", id), false, err)
		return nil, err
	}

	// Mock触发：发送订单已完成通知
	if s.notificationCallback != nil {
		go func() {
			_ = s.notificationCallback.OnOrderCompleted(ctx, order.WinnerID, id)
		}()
	}

	s.logger.LogOperationWithData(ctx, logger.OperationComplete, logger.ObjectOrder, fmt.Sprintf("%d", id),
		true, nil, map[string]interface{}{
			"order_id":    id,
			"auction_id":  order.AuctionID,
			"product_id":  order.ProductID,
			"winner_id":   order.WinnerID,
			"final_price": order.FinalPrice,
			"duration_ms": time.Since(start).Milliseconds(),
		}, nil)

	return s.orderDAO.GetByID(ctx, id)
}

// MockNotificationCallback Mock通知回调实现（用于测试）
type MockNotificationCallback struct{}

func (m *MockNotificationCallback) OnOrderPaid(ctx context.Context, userID int64, orderID int64) error {
	fmt.Printf("[Mock] Order %d paid notification sent to user %d\n", orderID, userID)
	return nil
}

func (m *MockNotificationCallback) OnOrderShipped(ctx context.Context, userID int64, orderID int64) error {
	fmt.Printf("[Mock] Order %d shipped notification sent to user %d\n", orderID, userID)
	return nil
}

func (m *MockNotificationCallback) OnOrderCompleted(ctx context.Context, userID int64, orderID int64) error {
	fmt.Printf("[Mock] Order %d completed notification sent to user %d\n", orderID, userID)
	return nil
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
