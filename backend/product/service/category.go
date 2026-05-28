package service

import (
	"context"
	"errors"
	"product-service/dao"
	"product-service/model"
)

// CategoryService 类别服务
type CategoryService struct {
	categoryDAO *dao.CategoryDAO
}

// NewCategoryService 创建类别服务
func NewCategoryService(categoryDAO *dao.CategoryDAO) *CategoryService {
	return &CategoryService{
		categoryDAO: categoryDAO,
	}
}

// Create 创建类别
func (s *CategoryService) Create(ctx context.Context, category *model.Category) error {
	// 检查Code是否已存在
	exists, err := s.categoryDAO.ExistsByCode(ctx, category.Code)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("类别代码已存在")
	}

	return s.categoryDAO.Create(ctx, category)
}

// GetByID 根据ID获取类别
func (s *CategoryService) GetByID(ctx context.Context, id int64) (*model.Category, error) {
	return s.categoryDAO.GetByID(ctx, id)
}

// List 获取类别列表
func (s *CategoryService) List(ctx context.Context, statusFilter *int) ([]model.Category, int64, error) {
	return s.categoryDAO.List(ctx, statusFilter)
}

// Update 更新类别
func (s *CategoryService) Update(ctx context.Context, id int64, updates map[string]interface{}) (*model.Category, error) {
	category, err := s.categoryDAO.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 如果要更新Code，检查是否已存在
	if newCode, ok := updates["code"].(string); ok && newCode != category.Code {
		exists, err := s.categoryDAO.ExistsByCode(ctx, newCode)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("类别代码已存在")
		}
	}

	// 更新字段
	if name, ok := updates["name"].(string); ok {
		category.Name = name
	}
	if desc, ok := updates["description"].(string); ok {
		category.Description = desc
	}
	if code, ok := updates["code"].(string); ok {
		category.Code = code
	}
	if sortOrder, ok := updates["sort_order"].(int); ok {
		category.SortOrder = sortOrder
	}
	if status, ok := updates["status"].(int); ok {
		category.Status = model.CategoryStatus(status)
	}

	err = s.categoryDAO.Update(ctx, category)
	if err != nil {
		return nil, err
	}

	return category, nil
}

// Delete 删除类别 (T022: 删除保护逻辑)
func (s *CategoryService) Delete(ctx context.Context, id int64) error {
	// 检查是否有商品关联
	count, err := s.categoryDAO.CountProductsByCategoryID(ctx, id)
	if err != nil {
		return err
	}

	if count > 0 {
		return errors.New("该类别下有商品，无法删除")
	}

	return s.categoryDAO.Delete(ctx, id)
}