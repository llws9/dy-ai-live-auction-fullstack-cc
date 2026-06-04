package service

import (
	"context"
	"testing"

	"product-service/dao"
	"product-service/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ProductTestSuite 测试套件
type ProductTestSuite struct {
	suite.Suite
	db         *gorm.DB
	productDAO *dao.ProductDAO
	ruleDAO    *dao.AuctionRuleDAO
	service    *ProductService
}

// SetupSuite 初始化测试套件
func (suite *ProductTestSuite) SetupSuite() {
	// 使用 SQLite 内存数据库
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(suite.T(), err)

	// 自动迁移
	err = db.AutoMigrate(&model.Product{}, &model.Category{}, &model.AuctionRule{}, &model.LiveStream{})
	assert.NoError(suite.T(), err)

	suite.db = db
	suite.productDAO = dao.NewProductDAO(db)
	suite.ruleDAO = dao.NewAuctionRuleDAO(db)
	suite.service = NewProductService(suite.productDAO, suite.ruleDAO, dao.NewLiveStreamDAO(db))
}

// TearDownSuite 清理测试套件
func (suite *ProductTestSuite) TearDownSuite() {
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()
}

// SetupTest 每个测试前清理数据
func (suite *ProductTestSuite) SetupTest() {
	suite.db.Exec("DELETE FROM products")
	suite.db.Exec("DELETE FROM categories")
	suite.db.Exec("DELETE FROM auction_rules")
}

// TestCreateProduct 测试创建商品
func (suite *ProductTestSuite) TestCreateProduct() {
	ctx := context.Background()

	// 测试成功创建
	req := &CreateProductRequest{
		Name:        "Test Product",
		Description: "Test Description",
		Images:      []string{"image1.jpg", "image2.jpg"},
	}

	product, err := suite.service.CreateProduct(ctx, req)

	suite.NoError(err)
	suite.NotNil(product)
	suite.NotZero(product.ID)
	suite.Equal(req.Name, product.Name)
	suite.Equal(req.Description, product.Description)
	suite.Equal(model.ProductStatusDraft, product.Status)
}

// TestCreateProduct_EmptyName 测试空名称
func (suite *ProductTestSuite) TestCreateProduct_EmptyName() {
	ctx := context.Background()

	req := &CreateProductRequest{
		Name:        "",
		Description: "Test Description",
	}

	product, err := suite.service.CreateProduct(ctx, req)

	suite.NoError(err)
	suite.NotNil(product)
	suite.Equal("", product.Name)
}

// TestGetProduct 测试获取商品
func (suite *ProductTestSuite) TestGetProduct() {
	ctx := context.Background()

	// 创建测试商品
	created, err := suite.service.CreateProduct(ctx, &CreateProductRequest{
		Name:        "Test Product",
		Description: "Test Description",
	})
	suite.NoError(err)

	// 获取商品
	product, err := suite.service.GetProduct(ctx, int64(created.ID))

	suite.NoError(err)
	suite.NotNil(product)
	suite.Equal(created.ID, product.ID)
	suite.Equal("Test Product", product.Name)
}

// TestGetProduct_NotFound 测试获取不存在的商品
func (suite *ProductTestSuite) TestGetProduct_NotFound() {
	ctx := context.Background()

	product, err := suite.service.GetProduct(ctx, 99999)

	suite.Error(err)
	suite.Nil(product)
	suite.Equal(gorm.ErrRecordNotFound, err)
}

// TestUpdateProduct 测试更新商品
func (suite *ProductTestSuite) TestUpdateProduct() {
	ctx := context.Background()

	// 创建测试商品
	created, err := suite.service.CreateProduct(ctx, &CreateProductRequest{
		Name:        "Original Name",
		Description: "Original Description",
	})
	suite.NoError(err)

	// 更新商品
	req := &UpdateProductRequest{
		Name:        "Updated Name",
		Description: "Updated Description",
	}

	product, err := suite.service.UpdateProduct(ctx, int64(created.ID), req)

	suite.NoError(err)
	suite.NotNil(product)
	suite.Equal("Updated Name", product.Name)
	suite.Equal("Updated Description", product.Description)
}

// TestUpdateProduct_PartialUpdate 测试部分更新
func (suite *ProductTestSuite) TestUpdateProduct_PartialUpdate() {
	ctx := context.Background()

	// 创建测试商品
	created, err := suite.service.CreateProduct(ctx, &CreateProductRequest{
		Name:        "Original Name",
		Description: "Original Description",
	})
	suite.NoError(err)

	// 只更新名称
	req := &UpdateProductRequest{
		Name: "Updated Name",
	}

	product, err := suite.service.UpdateProduct(ctx, int64(created.ID), req)

	suite.NoError(err)
	suite.Equal("Updated Name", product.Name)
	suite.Equal("Original Description", product.Description) // 描述不应改变
}

// TestUpdateProduct_NotFound 测试更新不存在的商品
func (suite *ProductTestSuite) TestUpdateProduct_NotFound() {
	ctx := context.Background()

	req := &UpdateProductRequest{
		Name: "Updated Name",
	}

	product, err := suite.service.UpdateProduct(ctx, 99999, req)

	suite.Error(err)
	suite.Nil(product)
}

// TestDeleteProduct 测试删除商品
func (suite *ProductTestSuite) TestDeleteProduct() {
	ctx := context.Background()

	// 创建测试商品
	created, err := suite.service.CreateProduct(ctx, &CreateProductRequest{
		Name: "Test Product",
	})
	suite.NoError(err)

	// 删除商品
	err = suite.service.DeleteProduct(ctx, int64(created.ID))
	suite.NoError(err)

	// 验证已删除
	_, err = suite.service.GetProduct(ctx, int64(created.ID))
	suite.Error(err)
	suite.Equal(gorm.ErrRecordNotFound, err)
}

// TestListProducts 测试商品列表
func (suite *ProductTestSuite) TestListProducts() {
	ctx := context.Background()

	// 创建多个商品
	for i := 1; i <= 25; i++ {
		_, err := suite.service.CreateProduct(ctx, &CreateProductRequest{
			Name: "Product",
		})
		suite.NoError(err)
	}

	// 测试分页
	products, total, err := suite.service.ListProducts(ctx, nil, 1, 10)

	suite.NoError(err)
	suite.Len(products, 10)
	suite.Equal(int64(25), total)

	// 测试第二页
	products, total, err = suite.service.ListProducts(ctx, nil, 2, 10)

	suite.NoError(err)
	suite.Len(products, 10)
	suite.Equal(int64(25), total)

	// 测试第三页
	products, total, err = suite.service.ListProducts(ctx, nil, 3, 10)

	suite.NoError(err)
	suite.Len(products, 5) // 只剩5个
	suite.Equal(int64(25), total)
}

// TestListProducts_DefaultPagination 测试默认分页
func (suite *ProductTestSuite) TestListProducts_DefaultPagination() {
	ctx := context.Background()

	// 创建商品
	for i := 1; i <= 5; i++ {
		_, err := suite.service.CreateProduct(ctx, &CreateProductRequest{
			Name: "Product",
		})
		suite.NoError(err)
	}

	// 使用默认分页
	products, total, err := suite.service.ListProducts(ctx, nil, 0, 0)

	suite.NoError(err)
	suite.Len(products, 5)
	suite.Equal(int64(5), total)
}

// TestPublishProduct 测试发布商品
func (suite *ProductTestSuite) TestPublishProduct() {
	ctx := context.Background()

	// 创建测试商品
	created, err := suite.service.CreateProduct(ctx, &CreateProductRequest{
		Name: "Test Product",
	})
	suite.NoError(err)

	// 发布商品
	_, _, err = suite.service.PublishProduct(ctx, int64(created.ID), 1, nil)
	suite.NoError(err)

	// 验证状态已更新
	product, err := suite.service.GetProduct(ctx, int64(created.ID))
	suite.NoError(err)
	suite.Equal(model.ProductStatusPublished, product.Status)
}

// TestCreateAuctionRule 测试创建竞拍规则
func (suite *ProductTestSuite) TestCreateAuctionRule() {
	ctx := context.Background()

	req := &CreateAuctionRuleRequest{
		ProductID: 1,
		Increment: 10.0,
		Duration:  3600,
	}

	rule, err := suite.service.CreateAuctionRule(ctx, req)

	suite.NoError(err)
	suite.NotNil(rule)
	suite.NotZero(rule.ID)
	suite.Equal(int64(1), rule.ProductID)
	suite.Equal(10.0, rule.Increment)
	suite.Equal(3600, rule.Duration)

	// 检查默认值
	suite.Equal(30, rule.DelayDuration)
	suite.Equal(180, rule.MaxDelayTime)
	suite.Equal(30, rule.TriggerDelayBefore)
}

// TestCreateAuctionRule_CustomValues 测试自定义竞拍规则
func (suite *ProductTestSuite) TestCreateAuctionRule_CustomValues() {
	ctx := context.Background()

	capPrice := 1000.0
	req := &CreateAuctionRuleRequest{
		ProductID:          1,
		StartPrice:         100.0,
		Increment:          5.0,
		CapPrice:           capPrice,
		Duration:           1800,
		DelayDuration:      60,
		MaxDelayTime:       300,
		TriggerDelayBefore: 45,
	}

	rule, err := suite.service.CreateAuctionRule(ctx, req)

	suite.NoError(err)
	suite.NotNil(rule)
	suite.Equal(100.0, rule.StartPrice)
	suite.Equal(5.0, rule.Increment)
	suite.Equal(capPrice, *rule.CapPrice)
	suite.Equal(60, rule.DelayDuration)
	suite.Equal(300, rule.MaxDelayTime)
	suite.Equal(45, rule.TriggerDelayBefore)
}

// TestGetAuctionRule 测试获取竞拍规则
func (suite *ProductTestSuite) TestGetAuctionRule() {
	ctx := context.Background()

	// 创建测试规则
	created, err := suite.service.CreateAuctionRule(ctx, &CreateAuctionRuleRequest{
		ProductID: 1,
		Increment: 10.0,
		Duration:  3600,
	})
	suite.NoError(err)

	// 获取规则
	rule, err := suite.service.GetAuctionRule(ctx, int64(created.ProductID))

	suite.NoError(err)
	suite.NotNil(rule)
	suite.Equal(created.ID, rule.ID)
}

// TestGetAuctionRule_NotFound 测试获取不存在的竞拍规则
func (suite *ProductTestSuite) TestGetAuctionRule_NotFound() {
	ctx := context.Background()

	rule, err := suite.service.GetAuctionRule(ctx, 99999)

	suite.NoError(err)
	suite.Nil(rule)
}

// TestRunSuite 运行测试套件
func TestRunSuite(t *testing.T) {
	suite.Run(t, new(ProductTestSuite))
}
