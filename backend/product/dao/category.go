package dao

import (
	"context"
	"product-service/model"

	"gorm.io/gorm"
)

// CategoryDAO 类别数据访问层
type CategoryDAO struct {
	db *gorm.DB
}

// NewCategoryDAO 创建类别DAO
func NewCategoryDAO(db *gorm.DB) *CategoryDAO {
	return &CategoryDAO{db: db}
}

// Create 创建类别
func (d *CategoryDAO) Create(ctx context.Context, category *model.Category) error {
	return d.db.WithContext(ctx).Create(category).Error
}

// GetByID 根据ID获取类别
func (d *CategoryDAO) GetByID(ctx context.Context, id int64) (*model.Category, error) {
	var category model.Category
	err := d.db.WithContext(ctx).First(&category, id).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// GetByCode 根据Code获取类别
func (d *CategoryDAO) GetByCode(ctx context.Context, code string) (*model.Category, error) {
	var category model.Category
	err := d.db.WithContext(ctx).Where("code = ?", code).First(&category).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// List 获取类别列表
func (d *CategoryDAO) List(ctx context.Context, statusFilter *int) ([]model.Category, int64, error) {
	var categories []model.Category
	var total int64

	query := d.db.WithContext(ctx).Model(&model.Category{})

	if statusFilter != nil {
		query = query.Where("status = ?", *statusFilter)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("sort_order ASC, created_at ASC").Find(&categories).Error
	if err != nil {
		return nil, 0, err
	}

	return categories, total, nil
}

// Update 更新类别
func (d *CategoryDAO) Update(ctx context.Context, category *model.Category) error {
	return d.db.WithContext(ctx).Save(category).Error
}

// Delete 删除类别
func (d *CategoryDAO) Delete(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Delete(&model.Category{}, id).Error
}

// CountProductsByCategoryID 统计类别下的商品数量
func (d *CategoryDAO) CountProductsByCategoryID(ctx context.Context, categoryID int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.Product{}).Where("category_id = ?", categoryID).Count(&count).Error
	return count, err
}

// ExistsByCode 检查Code是否已存在
func (d *CategoryDAO) ExistsByCode(ctx context.Context, code string) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.Category{}).Where("code = ?", code).Count(&count).Error
	return count > 0, err
}