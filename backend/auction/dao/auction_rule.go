package dao

import (
	"context"
	"errors"

	"auction-service/model"

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
