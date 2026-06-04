package service

import (
	"context"
	"testing"

	"product-service/dao"
	"product-service/model"

	"github.com/shopspring/decimal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// OrderTestSuite 订单测试套件
type OrderTestSuite struct {
	suite.Suite
	db         *gorm.DB
	orderDAO   *dao.OrderDAO
	historyDAO *dao.HistoryDAO
	service    *OrderService
}

// SetupSuite 初始化测试套件
func (suite *OrderTestSuite) SetupSuite() {
	// 使用 SQLite 内存数据库
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(suite.T(), err)

	// 自动迁移
	err = db.AutoMigrate(&model.Order{}, &model.Product{})
	assert.NoError(suite.T(), err)

	suite.db = db
	suite.orderDAO = dao.NewOrderDAO(db)
	suite.historyDAO = dao.NewHistoryDAO(db)
	suite.service = NewOrderService(suite.orderDAO, suite.historyDAO)
	suite.service.SetProductDAO(dao.NewProductDAO(db))
}

// TearDownSuite 清理测试套件
func (suite *OrderTestSuite) TearDownSuite() {
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()
}

// SetupTest 每个测试前清理数据
func (suite *OrderTestSuite) SetupTest() {
	suite.db.Exec("DELETE FROM orders")
	suite.db.Exec("DELETE FROM products")
}

// TestCreateOrder 测试创建订单
func (suite *OrderTestSuite) TestCreateOrder() {
	ctx := context.Background()

	order, err := suite.service.CreateOrder(ctx, 1, 1, 100, decimal.NewFromInt(500))

	suite.NoError(err)
	suite.NotNil(order)
	suite.NotZero(order.ID)
	suite.Equal(int64(1), order.AuctionID)
	suite.Equal(int64(1), order.ProductID)
	suite.Equal(int64(100), order.WinnerID)
	suite.Equal(decimal.NewFromInt(500), order.FinalPrice)
	suite.Equal(model.OrderStatusPending, order.Status)
}

func (suite *OrderTestSuite) TestCreateOrderStoresSellerIDFromProductOwner() {
	ctx := context.Background()
	ownerID := int64(1001)
	suite.NoError(suite.db.Create(&model.Product{
		ID:      10,
		OwnerID: &ownerID,
		Name:    "merchant product",
		Status:  model.ProductStatusPublished,
	}).Error)

	order, err := suite.service.CreateOrder(ctx, 101, 10, 2001, decimal.NewFromInt(500))

	suite.NoError(err)
	suite.NotNil(order.SellerID)
	suite.Equal(ownerID, *order.SellerID)
}

// TestGetOrder 测试获取订单
func (suite *OrderTestSuite) TestGetOrder() {
	ctx := context.Background()

	// 创建测试订单
	created, err := suite.service.CreateOrder(ctx, 1, 1, 100, decimal.NewFromInt(500))
	suite.NoError(err)

	// 获取订单
	order, err := suite.service.GetOrder(ctx, int64(created.ID))

	suite.NoError(err)
	suite.NotNil(order)
	suite.Equal(created.ID, order.ID)
	suite.Equal(int64(100), order.WinnerID)
}

// TestGetOrder_NotFound 测试获取不存在的订单
func (suite *OrderTestSuite) TestGetOrder_NotFound() {
	ctx := context.Background()

	order, err := suite.service.GetOrder(ctx, 99999)

	suite.Error(err)
	suite.Nil(order)
}

// TestListOrders 测试订单列表
func (suite *OrderTestSuite) TestListOrders() {
	ctx := context.Background()

	// 创建多个订单
	for i := 1; i <= 5; i++ {
		_, err := suite.service.CreateOrder(ctx, int64(i), int64(i), 100, decimal.NewFromInt(int64(i*100)))
		suite.NoError(err)
	}

	// 获取订单列表
	orders, total, err := suite.service.ListOrders(ctx, nil, 1, 10)

	suite.NoError(err)
	suite.Len(orders, 5)
	suite.Equal(int64(5), total)
}

// TestListOrders_ByUser 测试按用户获取订单
func (suite *OrderTestSuite) TestListOrders_ByUser() {
	ctx := context.Background()

	// 创建不同用户的订单
	_, err := suite.service.CreateOrder(ctx, 1, 1, 100, decimal.NewFromInt(100))
	suite.NoError(err)
	_, err = suite.service.CreateOrder(ctx, 2, 2, 100, decimal.NewFromInt(200))
	suite.NoError(err)
	_, err = suite.service.CreateOrder(ctx, 3, 3, 200, decimal.NewFromInt(300))
	suite.NoError(err)

	// 获取用户100的订单
	userID := int64(100)
	orders, total, err := suite.service.ListOrders(ctx, &userID, 1, 10)

	suite.NoError(err)
	suite.Len(orders, 2)
	suite.Equal(int64(2), total)
}

// TestPayOrder 测试支付订单
func (suite *OrderTestSuite) TestPayOrder() {
	ctx := context.Background()

	// 创建测试订单
	created, err := suite.service.CreateOrder(ctx, 1, 1, 100, decimal.NewFromInt(500))
	suite.NoError(err)
	suite.Equal(model.OrderStatusPending, created.Status)

	// 支付订单
	order, err := suite.service.PayOrder(ctx, int64(created.ID))

	suite.NoError(err)
	suite.NotNil(order)
	suite.Equal(model.OrderStatusPaid, order.Status)
}

// TestPayOrder_InvalidStatus 测试无效状态的订单支付
func (suite *OrderTestSuite) TestPayOrder_InvalidStatus() {
	ctx := context.Background()

	// 创建已支付的订单
	created, err := suite.service.CreateOrder(ctx, 1, 1, 100, decimal.NewFromInt(500))
	suite.NoError(err)

	// 第一次支付
	_, err = suite.service.PayOrder(ctx, int64(created.ID))
	suite.NoError(err)

	// 尝试再次支付（应该失败）
	order, err := suite.service.PayOrder(ctx, int64(created.ID))

	suite.Error(err)
	suite.Nil(order)
	suite.Contains(err.Error(), "订单状态不允许支付")
}

// TestShipOrder 测试发货
func (suite *OrderTestSuite) TestShipOrder() {
	ctx := context.Background()

	// 创建并支付订单
	created, err := suite.service.CreateOrder(ctx, 1, 1, 100, decimal.NewFromInt(500))
	suite.NoError(err)

	_, err = suite.service.PayOrder(ctx, int64(created.ID))
	suite.NoError(err)

	// 发货
	order, err := suite.service.ShipOrder(ctx, int64(created.ID))

	suite.NoError(err)
	suite.NotNil(order)
	suite.Equal(model.OrderStatusShipped, order.Status)
}

// TestShipOrder_InvalidStatus 测试无效状态的发货
func (suite *OrderTestSuite) TestShipOrder_InvalidStatus() {
	ctx := context.Background()

	// 创建待支付订单
	created, err := suite.service.CreateOrder(ctx, 1, 1, 100, decimal.NewFromInt(500))
	suite.NoError(err)

	// 尝试发货（应该失败）
	order, err := suite.service.ShipOrder(ctx, int64(created.ID))

	suite.Error(err)
	suite.Nil(order)
	suite.Contains(err.Error(), "订单状态不允许发货")
}

// TestCompleteOrder 测试完成订单
func (suite *OrderTestSuite) TestCompleteOrder() {
	ctx := context.Background()

	// 创建完整流程的订单
	created, err := suite.service.CreateOrder(ctx, 1, 1, 100, decimal.NewFromInt(500))
	suite.NoError(err)

	_, err = suite.service.PayOrder(ctx, int64(created.ID))
	suite.NoError(err)

	_, err = suite.service.ShipOrder(ctx, int64(created.ID))
	suite.NoError(err)

	// 完成订单
	order, err := suite.service.CompleteOrder(ctx, int64(created.ID))

	suite.NoError(err)
	suite.NotNil(order)
	suite.Equal(model.OrderStatusCompleted, order.Status)
}

// TestCompleteOrder_InvalidStatus 测试无效状态的完成
func (suite *OrderTestSuite) TestCompleteOrder_InvalidStatus() {
	ctx := context.Background()

	// 创建已支付但未发货的订单
	created, err := suite.service.CreateOrder(ctx, 1, 1, 100, decimal.NewFromInt(500))
	suite.NoError(err)

	_, err = suite.service.PayOrder(ctx, int64(created.ID))
	suite.NoError(err)

	// 尝试完成（应该失败）
	order, err := suite.service.CompleteOrder(ctx, int64(created.ID))

	suite.Error(err)
	suite.Nil(order)
	suite.Contains(err.Error(), "订单状态不允许完成")
}

// TestOrderStatusFlow 测试订单状态流转
func (suite *OrderTestSuite) TestOrderStatusFlow() {
	ctx := context.Background()

	// 完整的订单流程：创建 -> 支付 -> 发货 -> 完成
	order, err := suite.service.CreateOrder(ctx, 1, 1, 100, decimal.NewFromInt(500))
	suite.NoError(err)
	suite.Equal(model.OrderStatusPending, order.Status)

	// 支付
	order, err = suite.service.PayOrder(ctx, int64(order.ID))
	suite.NoError(err)
	suite.Equal(model.OrderStatusPaid, order.Status)

	// 发货
	order, err = suite.service.ShipOrder(ctx, int64(order.ID))
	suite.NoError(err)
	suite.Equal(model.OrderStatusShipped, order.Status)

	// 完成
	order, err = suite.service.CompleteOrder(ctx, int64(order.ID))
	suite.NoError(err)
	suite.Equal(model.OrderStatusCompleted, order.Status)
}

// TestNotificationCallback 测试通知回调
func (suite *OrderTestSuite) TestNotificationCallback() {
	ctx := context.Background()

	// 设置 Mock 通知回调
	suite.service.SetNotificationCallback(&MockNotificationCallback{})

	// 创建并支付订单
	created, err := suite.service.CreateOrder(ctx, 1, 1, 100, decimal.NewFromInt(500))
	suite.NoError(err)

	// 支付（应该触发通知回调）
	order, err := suite.service.PayOrder(ctx, int64(created.ID))
	suite.NoError(err)
	suite.Equal(model.OrderStatusPaid, order.Status)
}

// TestGetUserHistory 测试获取用户历史（降级路径：historyDAO=nil）
func (suite *OrderTestSuite) TestGetUserHistory() {
	ctx := context.Background()

	// 使用 nil historyDAO 构造的 service，走降级返回空列表
	svc := NewOrderService(suite.orderDAO, nil)
	items, total, err := svc.GetUserHistory(ctx, 100, 1, 10)

	suite.NoError(err)
	suite.NotNil(items)
	suite.Equal(int64(0), total)
}

func (suite *OrderTestSuite) TestGetSummary() {
	ctx := context.Background()
	userID := int64(100)

	_, err := suite.service.CreateOrder(ctx, 101, 1, userID, decimal.NewFromInt(500))
	suite.NoError(err)
	paid, err := suite.service.CreateOrder(ctx, 102, 1, userID, decimal.NewFromInt(600))
	suite.NoError(err)
	_, err = suite.service.PayOrder(ctx, paid.ID)
	suite.NoError(err)
	_, err = suite.service.CreateOrder(ctx, 103, 1, 200, decimal.NewFromInt(700))
	suite.NoError(err)

	summary, err := suite.service.GetSummary(ctx, userID)

	suite.NoError(err)
	suite.Equal(int64(1), summary.PendingPayment)
	suite.Equal(int64(1), summary.WonNotPaid)
}

// TestRunSuite 运行测试套件
func TestRunOrderSuite(t *testing.T) {
	suite.Run(t, new(OrderTestSuite))
}
