package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"product-service/dao"
	"product-service/model"
)

var ErrInvalidCategory = errors.New("invalid category_id")

// CreateProductRequest 创建商品请求
type CreateProductRequest struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Images      []string            `json:"images"`
	CategoryID  *int64              `json:"category_id"`
	Status      model.ProductStatus `json:"status,omitempty"`
}

// UpdateProductRequest 更新商品请求
type UpdateProductRequest struct {
	Name        string              `json:"name,omitempty"`
	Description string              `json:"description,omitempty"`
	Images      []string            `json:"images,omitempty"`
	CategoryID  *int64              `json:"category_id,omitempty"`
	Status      model.ProductStatus `json:"status,omitempty"`
}

// CreateAuctionRuleRequest 创建竞拍规则请求
type CreateAuctionRuleRequest struct {
	ProductID          int64   `json:"product_id"`
	StartPrice         float64 `json:"start_price"`
	Increment          float64 `json:"increment"`
	CapPrice           float64 `json:"cap_price,omitempty"`
	Duration           int     `json:"duration"`
	DelayDuration      int     `json:"delay_duration,omitempty"`
	MaxDelayTime       int     `json:"max_delay_time,omitempty"`
	TriggerDelayBefore int     `json:"trigger_delay_before,omitempty"`
}

// ProductService 商品服务
type ProductService struct {
	productDAO        *dao.ProductDAO
	ruleDAO           *dao.AuctionRuleDAO
	liveStreamDAO     *dao.LiveStreamDAO
	liveStreamService *LiveStreamService
}

// NewProductService 创建商品服务
func NewProductService(productDAO *dao.ProductDAO, ruleDAO *dao.AuctionRuleDAO, liveStreamDAO *dao.LiveStreamDAO) *ProductService {
	liveStreamService := NewLiveStreamService(liveStreamDAO)
	return &ProductService{
		productDAO:        productDAO,
		ruleDAO:           ruleDAO,
		liveStreamDAO:     liveStreamDAO,
		liveStreamService: liveStreamService,
	}
}

// PublishProduct 发布商品到直播间
func (s *ProductService) PublishProduct(ctx context.Context, productID, creatorID int64, startTime *time.Time) (*model.Product, *model.LiveStream, error) {
	// 1. 验证商品状态为草稿
	product, err := s.productDAO.GetByID(ctx, productID)
	if err != nil {
		return nil, nil, err
	}

	if product.Status != model.ProductStatusDraft {
		return nil, nil, errors.New("商品状态不正确，只有草稿状态的商品可以发布")
	}

	// 2. 获取或创建直播间
	liveStream, err := s.liveStreamDAO.GetOrCreateByCreatorID(ctx, creatorID, "")
	if err != nil {
		return nil, nil, err
	}

	// 3. 检查直播间状态
	if !liveStream.IsActive() {
		return nil, nil, errors.New("直播间已被禁用，无法发布商品")
	}

	// 4. 获取竞拍规则（已配置则验证，未配置将使用默认规则）
	_, err = s.ruleDAO.GetByProductID(ctx, productID)
	if err != nil {
		// 未配置规则不影响发布，auction-service会使用默认规则
	}

	// 5. 设置竞拍开始时间
	if startTime == nil {
		defaultStartTime := time.Now().Add(30 * time.Minute) // 默认30分钟后开始
		startTime = &defaultStartTime
	}

	// 6. 更新商品状态
	product.Status = model.ProductStatusPublished
	if err := s.productDAO.Update(ctx, product); err != nil {
		return nil, nil, err
	}

	// 注意：实际的竞拍记录创建将在 auction-service 中完成
	// 这里通过 HTTP 调用或消息队列通知 auction-service

	return product, liveStream, nil
}

// UnpublishProduct 下架商品
func (s *ProductService) UnpublishProduct(ctx context.Context, productID, creatorID int64, reason string) (*model.Product, error) {
	// 1. 验证商品状态为已发布
	product, err := s.productDAO.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	if product.Status != model.ProductStatusPublished {
		return nil, errors.New("商品状态不正确，只有已发布的商品可以下架")
	}

	// 2. 更新商品状态
	product.Status = model.ProductStatusUnpublished
	if err := s.productDAO.Update(ctx, product); err != nil {
		return nil, err
	}

	// 注意：取消竞拍记录和发送通知将在 auction-service 中完成
	// 这里通过 HTTP 调用或消息队列通知 auction-service

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

// CreateProduct 创建商品
func (s *ProductService) CreateProduct(ctx context.Context, req *CreateProductRequest) (*model.Product, error) {
	if err := s.validateCategoryID(ctx, req.CategoryID); err != nil {
		return nil, err
	}
	product := &model.Product{
		Name:        req.Name,
		Description: req.Description,
		Images:      req.Images,
		CategoryID:  req.CategoryID,
		Status:      model.ProductStatusDraft,
	}
	if err := s.productDAO.Create(ctx, product); err != nil {
		return nil, err
	}
	return product, nil
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
	if len(req.Images) > 0 {
		product.Images = req.Images
	}
	if req.CategoryID != nil {
		if err := s.validateCategoryID(ctx, req.CategoryID); err != nil {
			return nil, err
		}
		product.CategoryID = req.CategoryID
	}
	if req.Status != 0 {
		product.Status = req.Status
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

// CreateAuctionRule 创建竞拍规则
func (s *ProductService) CreateAuctionRule(ctx context.Context, req *CreateAuctionRuleRequest) (*model.AuctionRule, error) {
	rule := &model.AuctionRule{
		ProductID:          req.ProductID,
		StartPrice:         req.StartPrice,
		Increment:          req.Increment,
		Duration:           req.Duration,
		DelayDuration:      req.DelayDuration,
		MaxDelayTime:       req.MaxDelayTime,
		TriggerDelayBefore: req.TriggerDelayBefore,
	}
	if req.CapPrice > 0 {
		rule.CapPrice = &req.CapPrice
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

// MaxBatchProductIDs 是 GetProductsByIDs 单次允许的最大 id 数。
// 与 spec C §5.1.1 对齐：批量接口单次不超过 200 个。
const MaxBatchProductIDs = 200

// ListProductsByCategory 按 category_id 过滤商品（内部接口用）。
// page<=0 默认 1；pageSize<=0 默认 500；上限 1000，与 spec §5.1.2 对齐。
func (s *ProductService) ListProductsByCategory(ctx context.Context, categoryID int64, page, pageSize int) ([]model.Product, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 500
	}
	if pageSize > 1000 {
		pageSize = 1000
	}
	return s.productDAO.ListByCategoryID(ctx, categoryID, page, pageSize)
}

// GetProductsByIDs 按 id 列表批量获取商品（内部接口用）。
// - ids 为空/nil 直接返回空切片，不查 DB；
// - 超过 MaxBatchProductIDs 返回错误（spec C §5.1.1）；
// - 已删除/不存在的 id 不出现在结果中，由调用方按 id 自行 map。
func (s *ProductService) GetProductsByIDs(ctx context.Context, ids []int64) ([]model.Product, error) {
	if len(ids) == 0 {
		return []model.Product{}, nil
	}
	if len(ids) > MaxBatchProductIDs {
		return nil, errors.New("ids 数量超过上限")
	}
	return s.productDAO.GetByIDs(ctx, ids)
}

func (s *ProductService) validateCategoryID(ctx context.Context, categoryID *int64) error {
	if categoryID == nil {
		return nil
	}
	if _, err := s.productDAO.GetActiveCategoryByID(ctx, *categoryID); err != nil {
		return fmt.Errorf("%w: %d", ErrInvalidCategory, *categoryID)
	}
	return nil
}
