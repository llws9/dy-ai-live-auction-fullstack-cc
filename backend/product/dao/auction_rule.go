package dao

import (
	"context"
	"errors"

	"product-service/model"

	"gorm.io/gorm"
)

// AuctionRuleDAO 竞拍规则数据访问层
type AuctionRuleDAO struct {
	db *gorm.DB
}

// NewAuctionRuleDAO 创建竞拍规则 DAO
func NewAuctionRuleDAO(db *gorm.DB) *AuctionRuleDAO {
	return &AuctionRuleDAO{db: db}
}

// Create 创建竞拍规则
func (d *AuctionRuleDAO) Create(ctx context.Context, rule *model.AuctionRule) error {
	return d.db.WithContext(ctx).Create(rule).Error
}

// GetByID 根据 ID 获取规则
func (d *AuctionRuleDAO) GetByID(ctx context.Context, id int64) (*model.AuctionRule, error) {
	var rule model.AuctionRule
	err := d.db.WithContext(ctx).First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// GetByProductID 根据商品 ID 获取规则
func (d *AuctionRuleDAO) GetByProductID(ctx context.Context, productID int64) (*model.AuctionRule, error) {
	var rule model.AuctionRule
	err := d.db.WithContext(ctx).Where("product_id = ?", productID).First(&rule).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

// Update 更新竞拍规则
func (d *AuctionRuleDAO) Update(ctx context.Context, rule *model.AuctionRule) error {
	return d.db.WithContext(ctx).Save(rule).Error
}

// Delete 删除竞拍规则
func (d *AuctionRuleDAO) Delete(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Delete(&model.AuctionRule{}, id).Error
}

// Upsert 创建或更新竞拍规则
func (d *AuctionRuleDAO) Upsert(ctx context.Context, rule *model.AuctionRule) error {
	existing, err := d.GetByProductID(ctx, rule.ProductID)
	if err != nil {
		return err
	}
	if existing == nil {
		return d.Create(ctx, rule)
	}

	updates := map[string]interface{}{
		"start_price":          rule.StartPrice,
		"increment":            rule.Increment,
		"cap_price":            rule.CapPrice,
		"duration":             rule.Duration,
		"delay_duration":       rule.DelayDuration,
		"max_delay_time":       rule.MaxDelayTime,
		"trigger_delay_before": rule.TriggerDelayBefore,
	}
	if err := d.db.WithContext(ctx).
		Model(&model.AuctionRule{}).
		Where("id = ?", existing.ID).
		Updates(updates).Error; err != nil {
		return err
	}
	rule.ID = existing.ID
	rule.CreatedAt = existing.CreatedAt
	return nil
}
