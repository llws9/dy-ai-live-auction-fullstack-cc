package dao

import (
	"context"

	"product-service/model"

	"gorm.io/gorm"
)

// ProductDAO 商品数据访问层
type ProductDAO struct {
	db *gorm.DB
}

// NewProductDAO 创建商品 DAO
func NewProductDAO(db *gorm.DB) *ProductDAO {
	return &ProductDAO{db: db}
}

// Create 创建商品
func (d *ProductDAO) Create(ctx context.Context, product *model.Product) error {
	return d.db.WithContext(ctx).Create(product).Error
}

// GetByID 根据 ID 获取商品
func (d *ProductDAO) GetByID(ctx context.Context, id int64) (*model.Product, error) {
	var product model.Product
	err := d.db.WithContext(ctx).First(&product, id).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// List 获取商品列表
func (d *ProductDAO) List(ctx context.Context, status *model.ProductStatus, page, pageSize int) ([]model.Product, int64, error) {
	var products []model.Product
	var total int64

	query := d.db.WithContext(ctx).Model(&model.Product{})
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&products).Error; err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

// Update 更新商品
func (d *ProductDAO) Update(ctx context.Context, product *model.Product) error {
	return d.db.WithContext(ctx).Save(product).Error
}

// Delete 删除商品
func (d *ProductDAO) Delete(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Delete(&model.Product{}, id).Error
}

// UpdateStatus 更新商品状态
func (d *ProductDAO) UpdateStatus(ctx context.Context, id int64, status model.ProductStatus) error {
	return d.db.WithContext(ctx).
		Model(&model.Product{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// ListByCategoryID 按 category_id 过滤商品列表（内部接口用）。
func (d *ProductDAO) ListByCategoryID(ctx context.Context, categoryID int64, page, pageSize int) ([]model.Product, int64, error) {
	var products []model.Product
	var total int64

	query := d.db.WithContext(ctx).Model(&model.Product{}).Where("category_id = ?", categoryID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id ASC").Find(&products).Error; err != nil {
		return nil, 0, err
	}
	return products, total, nil
}

// GetByIDs 按 id 列表批量获取商品（内部接口用，缺失 id 不出现在结果中）。
func (d *ProductDAO) GetByIDs(ctx context.Context, ids []int64) ([]model.Product, error) {
	if len(ids) == 0 {
		return []model.Product{}, nil
	}
	var products []model.Product
	if err := d.db.WithContext(ctx).Where("id IN ?", ids).Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}
