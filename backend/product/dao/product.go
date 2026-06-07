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
	err := d.db.WithContext(ctx).
		Model(&model.Product{}).
		Select("products.*, categories.name AS category_name").
		Joins("LEFT JOIN categories ON categories.id = products.category_id").
		Where("products.id = ?", id).
		First(&product).Error
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
	readQuery := d.db.WithContext(ctx).
		Model(&model.Product{}).
		Select("products.*, categories.name AS category_name").
		Joins("LEFT JOIN categories ON categories.id = products.category_id")
	if status != nil {
		readQuery = readQuery.Where("products.status = ?", *status)
	}
	if err := readQuery.
		Offset(offset).
		Limit(pageSize).
		Order("products.created_at DESC").
		Find(&products).Error; err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

// ListAdminScoped returns all products for admins, or only owner products for merchants.
func (d *ProductDAO) ListAdminScoped(ctx context.Context, ownerID *int64, status *model.ProductStatus, page, pageSize int) ([]model.Product, int64, error) {
	var products []model.Product
	var total int64

	query := d.db.WithContext(ctx).Model(&model.Product{})
	if ownerID != nil {
		query = query.Where("owner_id = ?", *ownerID)
	}
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

// ListAdminScopedAll returns all admin-visible products before application-level derived status filtering.
func (d *ProductDAO) ListAdminScopedAll(ctx context.Context, ownerID *int64) ([]model.Product, error) {
	var products []model.Product
	query := d.db.WithContext(ctx).Model(&model.Product{})
	if ownerID != nil {
		query = query.Where("owner_id = ?", *ownerID)
	}
	if err := query.Order("created_at DESC").Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

// GetByIDAndOwnerID returns a product only when it belongs to ownerID.
func (d *ProductDAO) GetByIDAndOwnerID(ctx context.Context, id, ownerID int64) (*model.Product, error) {
	var product model.Product
	err := d.db.WithContext(ctx).Where("id = ? AND owner_id = ?", id, ownerID).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// Update 更新商品
func (d *ProductDAO) Update(ctx context.Context, product *model.Product) error {
	return d.db.WithContext(ctx).Save(product).Error
}

// Delete 删除商品
func (d *ProductDAO) Delete(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Delete(&model.Product{}, id).Error
}

// DeleteByIDAndOwnerID deletes a product only when it belongs to ownerID.
func (d *ProductDAO) DeleteByIDAndOwnerID(ctx context.Context, id, ownerID int64) error {
	return d.db.WithContext(ctx).Where("id = ? AND owner_id = ?", id, ownerID).Delete(&model.Product{}).Error
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

	query := d.db.WithContext(ctx).Model(&model.Product{}).
		Where("category_id = ? AND status = ?", categoryID, model.ProductStatusPublished)
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
	if err := d.db.WithContext(ctx).
		Where("id IN ? AND status = ?", ids, model.ProductStatusPublished).
		Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

// GetActiveCategoryByID 获取启用中的类别。
func (d *ProductDAO) GetActiveCategoryByID(ctx context.Context, id int64) (*model.Category, error) {
	var category model.Category
	err := d.db.WithContext(ctx).
		Where("id = ? AND status = ?", id, model.CategoryStatusActive).
		First(&category).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}
