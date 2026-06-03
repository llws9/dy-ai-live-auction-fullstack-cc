package dao

import (
	"context"
	"errors"

	"auction-service/model"

	"gorm.io/gorm"
)

// ErrIllegalStatusTransition 非法的一口价商品状态流转
var ErrIllegalStatusTransition = errors.New("illegal fixed price status transition")

// legalFixedPriceTransitions 合法状态流转表
var legalFixedPriceTransitions = map[model.FixedPriceStatus]map[model.FixedPriceStatus]bool{
	model.FixedPriceStatusOnSale: {
		model.FixedPriceStatusSoldOut: true,
		model.FixedPriceStatusOffline: true,
	},
	model.FixedPriceStatusSoldOut: {
		model.FixedPriceStatusOffline: true,
	},
	model.FixedPriceStatusOffline: {},
}

// FixedPriceItemDAO 一口价商品数据访问层
type FixedPriceItemDAO struct {
	db *gorm.DB
}

// NewFixedPriceItemDAO 创建一口价商品 DAO
func NewFixedPriceItemDAO(db *gorm.DB) *FixedPriceItemDAO {
	return &FixedPriceItemDAO{db: db}
}

// Create 创建一口价商品
func (d *FixedPriceItemDAO) Create(ctx context.Context, item *model.FixedPriceItem) error {
	return d.db.WithContext(ctx).Create(item).Error
}

// GetByID 根据 ID 获取一口价商品
func (d *FixedPriceItemDAO) GetByID(ctx context.Context, id int64) (*model.FixedPriceItem, error) {
	var item model.FixedPriceItem
	if err := d.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdateStatus 校验合法流转后更新状态
func (d *FixedPriceItemDAO) UpdateStatus(ctx context.Context, id int64, to model.FixedPriceStatus) error {
	item, err := d.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !legalFixedPriceTransitions[item.Status][to] {
		return ErrIllegalStatusTransition
	}
	return d.db.WithContext(ctx).
		Model(&model.FixedPriceItem{}).
		Where("id = ?", id).
		Update("status", to).Error
}

// ListByLiveStreamID 按直播间查询一口价商品，可选状态过滤，按创建时间倒序
func (d *FixedPriceItemDAO) ListByLiveStreamID(ctx context.Context, liveStreamID int64, statuses []model.FixedPriceStatus) ([]*model.FixedPriceItem, error) {
	var items []*model.FixedPriceItem
	q := d.db.WithContext(ctx).Where("live_stream_id = ?", liveStreamID)
	if len(statuses) > 0 {
		q = q.Where("status IN ?", statuses)
	}
	if err := q.Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// DecrementRemainingStock 更新剩余库存（供异步兜底使用）
func (d *FixedPriceItemDAO) DecrementRemainingStock(ctx context.Context, id int64, newRemaining int) error {
	return d.db.WithContext(ctx).
		Model(&model.FixedPriceItem{}).
		Where("id = ?", id).
		Update("remaining_stock", newRemaining).Error
}
