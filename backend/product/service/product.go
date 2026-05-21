package service

import (
	"context"
	"errors"

	"product-service/dao"
	"product-service/model"
)

// ProductService 商品服务
type ProductService struct {
	productDAO *dao.ProductDAO
	ruleDAO    *dao.AuctionRuleDAO
}

// NewProductService 创建商品服务
func NewProductService(productDAO *dao.ProductDAO, ruleDAO *dao.AuctionRuleDAO) *ProductService {
	return &ProductService{
		productDAO: productDAO,
		ruleDAO:    ruleDAO,
	}
}

// CreateProductRequest 创建商品请求
type CreateProductRequest struct {
	Name        string   `json:"name" binding:"required,max=128"`
	Description string   `json:"description"`
	Images      []string `json:"images"`
}

// UpdateProductRequest 更新商品请求
type UpdateProductRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Images      []string `json:"images"`
}

// CreateProduct 创建商品
func (s *ProductService) CreateProduct(ctx context.Context, req *CreateProductRequest) (*model.Product, error) {
	if req.Name == "" {
		return nil, errors.New("商品名称不能为空")
	}

	product := &model.Product{
		Name:        req.Name,
		Description: req.Description,
		Images:      req.Images,
		Status:      model.ProductStatusDraft,
	}

	if err := s.productDAO.Create(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
}

// GetProduct 获取商品详情
func (s *ProductService) GetProduct(ctx context.Context, id int64) (*model.Product, error) {
	return s.productDAO.GetByID(ctx, id)
}

// ListProducts 获取商品列表
func (s *ProductService) ListProducts(ctx context.Context, status *model.ProductStatus, page, pageSize int) ([]model.Product, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	return s.productDAO.List(ctx, status, page, pageSize)
}

// UpdateProduct 更新商品
func (s *ProductService) UpdateProduct(ctx context.Context, id int64, req *UpdateProductRequest) (*model.Product, error) {
	product, err := s.productDAO.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Description != "" {
		product.Description = req.Description
	}
	if req.Images != nil {
		product.Images = req.Images
	}

	if err := s.productDAO.Update(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
}

// DeleteProduct 删除商品
func (s *ProductService) DeleteProduct(ctx context.Context, id int64) error {
	return s.productDAO.Delete(ctx, id)
}

// PublishProduct 发布商品
func (s *ProductService) PublishProduct(ctx context.Context, id int64) error {
	return s.productDAO.UpdateStatus(ctx, id, model.ProductStatusPublished)
}

// CreateAuctionRuleRequest 创建竞拍规则请求
type CreateAuctionRuleRequest struct {
	ProductID         int64    `json:"product_id"`
	StartPrice        float64  `json:"start_price"`                     // 默认 0
	Increment         float64  `json:"increment" binding:"required,gt=0"` // 加价幅度
	CapPrice         *float64  `json:"cap_price"`                       // 封顶价
	Duration          int      `json:"duration" binding:"required,gt=0"` // 竞拍时长（秒）
	DelayDuration     int      `json:"delay_duration"`                  // 单次延时时长，默认30秒
	MaxDelayTime      int      `json:"max_delay_time"`                  // 最大延时时长，默认180秒
	TriggerDelayBefore int      `json:"trigger_delay_before"`            // 延时触发时间，默认30秒
}

// CreateAuctionRule 创建竞拍规则
func (s *ProductService) CreateAuctionRule(ctx context.Context, req *CreateAuctionRuleRequest) (*model.AuctionRule, error) {
	// 设置默认值
	if req.DelayDuration == 0 {
		req.DelayDuration = 30
	}
	if req.MaxDelayTime == 0 {
		req.MaxDelayTime = 180
	}
	if req.TriggerDelayBefore == 0 {
		req.TriggerDelayBefore = 30
	}

	rule := &model.AuctionRule{
		ProductID:          req.ProductID,
		StartPrice:         req.StartPrice,
		Increment:          req.Increment,
		CapPrice:          req.CapPrice,
		Duration:           req.Duration,
		DelayDuration:      req.DelayDuration,
		MaxDelayTime:       req.MaxDelayTime,
		TriggerDelayBefore: req.TriggerDelayBefore,
	}

	if err := s.ruleDAO.Create(ctx, rule); err != nil {
		return nil, err
	}

	return rule, nil
}

// GetAuctionRule 获取竞拍规则
func (s *ProductService) GetAuctionRule(ctx context.Context, productID int64) (*model.AuctionRule, error) {
	return s.ruleDAO.GetByProductID(ctx, productID)
}
