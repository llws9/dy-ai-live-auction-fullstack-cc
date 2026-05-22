package main

import (
	"log"

	"github.com/cloudwego/hertz/pkg/app/server"

	"product-service/dao"
	"product-service/handler"
	"product-service/model"
	"product-service/service"
)

func main() {
	// 初始化数据库连接
	db, err := dao.InitDBFromEnv()
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}

	// 自动迁移表结构（如果表已存在，忽略错误）
	if err := db.AutoMigrate(
		&model.User{},
		&model.Product{},
		&model.AuctionRule{},
		&model.Order{},
	); err != nil {
		log.Printf("Warning: AutoMigrate failed (tables may already exist): %v", err)
	}

	// 初始化 DAO 层
	productDAO := dao.NewProductDAO(db)
	ruleDAO := dao.NewAuctionRuleDAO(db)
	orderDAO := dao.NewOrderDAO(db)
	historyDAO := dao.NewHistoryDAO(db)

	// 初始化 Service 层
	productService := service.NewProductService(productDAO, ruleDAO)
	orderService := service.NewOrderService(orderDAO, historyDAO)

	// 初始化 Handler 层
	productHandler := handler.NewProductHandler(productService)
	ruleHandler := handler.NewRuleHandler(productService)
	orderHandler := handler.NewOrderHandler(orderService)

	// 创建 Hertz 服务器
	h := server.Default(
		server.WithHostPorts(":8081"),
	)

	// 注册路由
	registerRoutes(h, productHandler, ruleHandler, orderHandler)

	// 启动服务
	log.Println("Product service starting on :8081")
	h.Spin()
}

// registerRoutes 注册路由
func registerRoutes(h *server.Hertz, productHandler *handler.ProductHandler, ruleHandler *handler.RuleHandler, orderHandler *handler.OrderHandler) {
	v1 := h.Group("/api/v1")

	// 商品相关路由
	v1.GET("/products", productHandler.List)
	v1.GET("/products/:id", productHandler.Get)
	v1.POST("/products", productHandler.Create)
	v1.PUT("/products/:id", productHandler.Update)
	v1.DELETE("/products/:id", productHandler.Delete)

	// 竞拍规则相关路由
	v1.POST("/products/:id/rules", ruleHandler.Create)
	v1.GET("/products/:id/rules", ruleHandler.Get)

	// 订单相关路由
	v1.GET("/orders", orderHandler.List)
	v1.GET("/orders/:id", orderHandler.Get)
	v1.PUT("/orders/:id", orderHandler.Update)
	v1.PUT("/orders/:id/pay", orderHandler.Pay)
	v1.PUT("/orders/:id/ship", orderHandler.Ship)
	v1.GET("/orders/history", orderHandler.GetUserHistory)
}
