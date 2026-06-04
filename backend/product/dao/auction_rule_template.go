package dao

import (
	"context"

	"product-service/model"

	"gorm.io/gorm"
)

type AuctionRuleTemplateDAO struct {
	db *gorm.DB
}

func NewAuctionRuleTemplateDAO(db *gorm.DB) *AuctionRuleTemplateDAO {
	return &AuctionRuleTemplateDAO{db: db}
}

func (d *AuctionRuleTemplateDAO) ListByOwner(ctx context.Context, ownerID int64, page, pageSize int) ([]model.AuctionRuleTemplate, int64, error) {
	var items []model.AuctionRuleTemplate
	var total int64
	query := d.db.WithContext(ctx).Model(&model.AuctionRuleTemplate{}).Where("owner_id = ?", ownerID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("is_default DESC, updated_at DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (d *AuctionRuleTemplateDAO) GetByIDAndOwner(ctx context.Context, id, ownerID int64) (*model.AuctionRuleTemplate, error) {
	var item model.AuctionRuleTemplate
	if err := d.db.WithContext(ctx).Where("id = ? AND owner_id = ?", id, ownerID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *AuctionRuleTemplateDAO) Create(ctx context.Context, item *model.AuctionRuleTemplate) error {
	return d.db.WithContext(ctx).Create(item).Error
}

func (d *AuctionRuleTemplateDAO) Update(ctx context.Context, item *model.AuctionRuleTemplate) error {
	return d.db.WithContext(ctx).Save(item).Error
}

func (d *AuctionRuleTemplateDAO) DeleteByIDAndOwner(ctx context.Context, id, ownerID int64) error {
	return d.db.WithContext(ctx).Where("id = ? AND owner_id = ?", id, ownerID).Delete(&model.AuctionRuleTemplate{}).Error
}
