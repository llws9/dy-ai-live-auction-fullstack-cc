package dao

import (
	"context"

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
