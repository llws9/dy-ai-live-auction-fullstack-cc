package dao

import (
	"context"
	"time"

	"gorm.io/gorm"

	"product-service/model"
)

// OrderDAO 订单数据访问层
type OrderDAO struct {
	db *gorm.DB
}

// NewOrderDAO 创建订单 DAO
func NewOrderDAO(db *gorm.DB) *OrderDAO {
	return &OrderDAO{db: db}
}

// Create 创建订单
func (d *OrderDAO) Create(ctx context.Context, order *model.Order) error {
	return d.db.WithContext(ctx).Create(order).Error
}

// GetByID 根据 ID 获取订单
func (d *OrderDAO) GetByID(ctx context.Context, id int64) (*model.Order, error) {
	var order model.Order
	err := d.db.WithContext(ctx).First(&order, id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (d *OrderDAO) GetByIDAndWinnerID(ctx context.Context, id, winnerID int64) (*model.Order, error) {
	var order model.Order
	err := d.db.WithContext(ctx).
		Where("id = ? AND winner_id = ?", id, winnerID).
		First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (d *OrderDAO) GetByIDAndSellerID(ctx context.Context, id, sellerID int64) (*model.Order, error) {
	var order model.Order
	err := d.db.WithContext(ctx).
		Where("id = ? AND seller_id = ?", id, sellerID).
		First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// List 获取订单列表
func (d *OrderDAO) List(ctx context.Context, userID *int64, page, pageSize int) ([]model.Order, int64, error) {
	var orders []model.Order
	var total int64

	query := d.db.WithContext(ctx).Model(&model.Order{})
	if userID != nil {
		query = query.Where("winner_id = ?", *userID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// CountByWinnerAndStatus 统计指定用户指定状态的订单数量
func (d *OrderDAO) CountByWinnerAndStatus(ctx context.Context, winnerID int64, status model.OrderStatus) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.Order{}).
		Where("winner_id = ? AND status = ?", winnerID, status).
		Count(&count).Error
	return count, err
}

// UpdateStatus 更新订单状态
func (d *OrderDAO) UpdateStatus(ctx context.Context, id int64, status model.OrderStatus) error {
	return d.db.WithContext(ctx).
		Model(&model.Order{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// GetByAuctionID 根据竞拍 ID 获取订单
func (d *OrderDAO) GetByAuctionID(ctx context.Context, auctionID int64) (*model.Order, error) {
	var order model.Order
	err := d.db.WithContext(ctx).Where("auction_id = ?", auctionID).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// GetList 获取订单列表（支持状态筛选）
func (d *OrderDAO) GetList(ctx context.Context, status *model.OrderStatus, page, pageSize int) ([]model.Order, int64, error) {
	var orders []model.Order
	var total int64

	query := d.db.WithContext(ctx).Model(&model.Order{})
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// Update 更新订单
func (d *OrderDAO) Update(ctx context.Context, order *model.Order) error {
	return d.db.WithContext(ctx).Save(order).Error
}

// PayOrder 支付订单
func (d *OrderDAO) PayOrder(ctx context.Context, id int64) error {
	now := time.Now()
	return d.db.WithContext(ctx).
		Model(&model.Order{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":  model.OrderStatusPaid,
			"paid_at": now,
		}).Error
}

// ShipOrder 发货
func (d *OrderDAO) ShipOrder(ctx context.Context, id int64) error {
	now := time.Now()
	return d.db.WithContext(ctx).
		Model(&model.Order{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     model.OrderStatusShipped,
			"shipped_at": now,
		}).Error
}

func (d *OrderDAO) ShipOrderForSeller(ctx context.Context, id, sellerID int64) error {
	now := time.Now()
	return d.db.WithContext(ctx).
		Model(&model.Order{}).
		Where("id = ? AND seller_id = ?", id, sellerID).
		Updates(map[string]interface{}{
			"status":     model.OrderStatusShipped,
			"shipped_at": now,
		}).Error
}

// CompleteOrder 完成订单
func (d *OrderDAO) CompleteOrder(ctx context.Context, id int64) error {
	now := time.Now()
	return d.db.WithContext(ctx).
		Model(&model.Order{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       model.OrderStatusCompleted,
			"completed_at": now,
		}).Error
}
