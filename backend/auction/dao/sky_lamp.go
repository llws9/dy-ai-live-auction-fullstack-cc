package dao

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"auction-service/model"
)

// SkyLampDAO 天灯订阅数据访问层
type SkyLampDAO struct {
	db *gorm.DB
}

// NewSkyLampDAO 创建天灯订阅 DAO
func NewSkyLampDAO(db *gorm.DB) *SkyLampDAO {
	return &SkyLampDAO{db: db}
}

// Create 创建订阅
func (d *SkyLampDAO) Create(ctx context.Context, subscription *model.SkyLampSubscription) error {
	return d.db.WithContext(ctx).Create(subscription).Error
}

// GetByID 根据ID获取订阅
func (d *SkyLampDAO) GetByID(ctx context.Context, id int64) (*model.SkyLampSubscription, error) {
	var subscription model.SkyLampSubscription
	err := d.db.WithContext(ctx).First(&subscription, id).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

// GetActiveByUser 获取用户在某个竞拍的活跃订阅
func (d *SkyLampDAO) GetActiveByUser(ctx context.Context, auctionID, userID int64) (*model.SkyLampSubscription, error) {
	var subscription model.SkyLampSubscription
	err := d.db.WithContext(ctx).
		Where("auction_id = ? AND user_id = ? AND status = ?", auctionID, userID, model.SkyLampStatusActive).
		First(&subscription).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &subscription, nil
}

// GetActiveByAuction 获取某个竞拍的所有活跃订阅（按创建时间升序，保证先订阅先跟价）
func (d *SkyLampDAO) GetActiveByAuction(ctx context.Context, auctionID int64) ([]model.SkyLampSubscription, error) {
	var subscriptions []model.SkyLampSubscription
	err := d.db.WithContext(ctx).
		Where("auction_id = ? AND status = ?", auctionID, model.SkyLampStatusActive).
		Order("id ASC").
		Find(&subscriptions).Error
	return subscriptions, err
}

// Update 更新订阅
func (d *SkyLampDAO) Update(ctx context.Context, subscription *model.SkyLampSubscription) error {
	return d.db.WithContext(ctx).Save(subscription).Error
}

// UpdateStatus 更新订阅状态
func (d *SkyLampDAO) UpdateStatus(ctx context.Context, id int64, status model.SkyLampStatus) error {
	return d.db.WithContext(ctx).
		Model(&model.SkyLampSubscription{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// UpdateStatusWithStoppedAt 更新订阅状态并写入停止时间
func (d *SkyLampDAO) UpdateStatusWithStoppedAt(ctx context.Context, id int64, status model.SkyLampStatus) error {
	now := time.Now()
	return d.db.WithContext(ctx).
		Model(&model.SkyLampSubscription{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"stopped_at": &now,
		}).Error
}

// StopByAuction 竞拍结束时停止所有订阅
func (d *SkyLampDAO) StopByAuction(ctx context.Context, auctionID int64) error {
	now := time.Now()
	return d.db.WithContext(ctx).
		Model(&model.SkyLampSubscription{}).
		Where("auction_id = ? AND status = ?", auctionID, model.SkyLampStatusActive).
		Updates(map[string]interface{}{
			"status":     model.SkyLampStatusEnded,
			"stopped_at": &now,
		}).Error
}

// GetByUserStatus 获取用户指定状态的订阅列表
func (d *SkyLampDAO) GetByUserStatus(ctx context.Context, userID int64, status model.SkyLampStatus, page, pageSize int) ([]model.SkyLampSubscription, int64, error) {
	var subscriptions []model.SkyLampSubscription
	var total int64

	query := d.db.WithContext(ctx).Model(&model.SkyLampSubscription{}).
		Where("user_id = ? AND status = ?", userID, status)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("id DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&subscriptions).Error

	return subscriptions, total, err
}

// Delete 删除订阅（用于首次出价失败回滚）
func (d *SkyLampDAO) Delete(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Delete(&model.SkyLampSubscription{}, id).Error
}