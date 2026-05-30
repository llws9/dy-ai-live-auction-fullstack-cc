package dao

import (
	"context"

	"gorm.io/gorm"

	"test-service/model"
)

// SeedDAO test_seed_data 表读写
type SeedDAO struct {
	db *gorm.DB
}

// NewSeedDAO 构造
func NewSeedDAO(db *gorm.DB) *SeedDAO {
	return &SeedDAO{db: db}
}

// Add 记录一条创建的业务实体
func (d *SeedDAO) Add(ctx context.Context, testID, kind string, refID int64) error {
	return d.db.WithContext(ctx).Create(&model.TestSeedData{
		TestID: testID,
		Kind:   kind,
		RefID:  refID,
	}).Error
}

// ListByTestID 列出某次测试创建的全部实体
func (d *SeedDAO) ListByTestID(ctx context.Context, testID string) ([]model.TestSeedData, error) {
	var list []model.TestSeedData
	if err := d.db.WithContext(ctx).Where("test_id = ?", testID).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// DeleteByTestID 清理某次测试的全部种子数据 ref（仅删 ref 表，业务表清理由调用方负责）
func (d *SeedDAO) DeleteByTestID(ctx context.Context, testID string) error {
	return d.db.WithContext(ctx).Where("test_id = ?", testID).Delete(&model.TestSeedData{}).Error
}
