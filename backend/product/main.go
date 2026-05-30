package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"

	"product-service/client"
	"product-service/config"
	"product-service/dao"
	"product-service/handler"
	"product-service/middleware"
	"product-service/model"
	"product-service/service"
)

func main() {
	// 从 Nacos 加载配置（失败时使用环境变量）
	cfg, nacosLoader := config.LoadFromNacosWithFallback()

	// 初始化数据库连接
	dbCfg := &dao.Config{
		Host:         cfg.Database.Host,
		Port:         "3306",
		User:         cfg.Database.User,
		Password:     cfg.Database.Password,
		Database:     cfg.Database.Name,
		MaxIdleConns: cfg.Database.MaxIdleConns,
		MaxOpenConns: cfg.Database.MaxOpenConns,
	}
	if cfg.Database.Port > 0 {
		dbCfg.Port = fmt.Sprintf("%d", cfg.Database.Port)
	}

	db, err := dao.InitDB(dbCfg)
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}

	// 自动迁移表结构（如果表已存在，忽略错误）
	if err := db.AutoMigrate(
		&model.User{},
		&model.Product{},
		&model.Category{},
		&model.AuctionRule{},
		&model.Order{},
		&model.LiveStream{},
	); err != nil {
		log.Printf("Warning: AutoMigrate failed (tables may already exist): %v", err)
	}

	// 初始化 DAO 层
	productDAO := dao.NewProductDAO(db)
	ruleDAO := dao.NewAuctionRuleDAO(db)
	orderDAO := dao.NewOrderDAO(db)
	historyDAO := dao.NewHistoryDAO(db)
	statisticsDAO := dao.NewStatisticsDAO(db)
	liveStreamDAO := dao.NewLiveStreamDAO(db)
	categoryDAO := dao.NewCategoryDAO(db)

	// 初始化 Service 层
	productService := service.NewProductService(productDAO, ruleDAO, liveStreamDAO)
	orderService := service.NewOrderService(orderDAO, historyDAO)
	statisticsService := service.NewStatisticsService(statisticsDAO)
	liveStreamService := service.NewLiveStreamService(liveStreamDAO)
	categoryService := service.NewCategoryService(categoryDAO)

	// 初始化 Handler 层
	productHandler := handler.NewProductHandler(productService)
	ruleHandler := handler.NewRuleHandler(productService)
	orderHandler := handler.NewOrderHandler(orderService)
	statisticsHandler := handler.NewStatisticsHandler(statisticsService)
	productPublishHandler := handler.NewProductHandler(productService)
	liveStreamHandler := handler.NewLiveStreamHandler(liveStreamService)
	auctionSvcURL := os.Getenv("AUCTION_SERVICE_URL")
	if auctionSvcURL == "" {
		auctionSvcURL = "http://localhost:8082"
	}
	liveStreamHandler.SetAuctionClient(client.NewAuctionClient(auctionSvcURL, 2*time.Second))
	categoryHandler := handler.NewCategoryHandler(categoryService)
	internalHandler := handler.NewInternalHandler(productService, liveStreamDAO)

	// 监听配置变更（如果 Nacos 可用）
	if nacosLoader != nil {
		go func() {
			_ = nacosLoader.LoadAndListen(cfg, func(newCfg interface{}) {
				log.Printf("Product config updated from Nacos")
			})
		}()
	}

	// 创建 Hertz 服务器
	h := server.Default(
		server.WithHostPorts(cfg.Server.Port),
	)

	// 注册路由
	registerRoutes(h, productHandler, ruleHandler, orderHandler, statisticsHandler, productPublishHandler, liveStreamHandler, categoryHandler, internalHandler)

	// 启动服务
	log.Printf("Product service starting on %s", cfg.Server.Port)
	h.Spin()
}

// registerRoutes 注册路由
func registerRoutes(h *server.Hertz, productHandler *handler.ProductHandler, ruleHandler *handler.RuleHandler, orderHandler *handler.OrderHandler, statisticsHandler *handler.StatisticsHandler, productPublishHandler *handler.ProductHandler, liveStreamHandler *handler.LiveStreamHandler, categoryHandler *handler.CategoryHandler, internalHandler *handler.InternalHandler) {
	v1 := h.Group("/api/v1")

	// 商品相关路由
	v1.GET("/products", productHandler.List)
	v1.GET("/products/:id", productHandler.Get)
	v1.POST("/products", productHandler.Create)
	v1.PUT("/products/:id", productHandler.Update)
	v1.DELETE("/products/:id", productHandler.Delete)

	// 商品发布/下架路由
	v1.POST("/products/:id/publish", productPublishHandler.PublishHandler)
	v1.POST("/products/:id/unpublish", productPublishHandler.UnpublishHandler)

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

	// 直播间相关路由
	v1.GET("/admin/live-streams", liveStreamHandler.ListAdmin)
	v1.GET("/live-streams/:id", liveStreamHandler.GetDetail)

	// 统计相关路由
	v1.GET("/statistics/overview", statisticsHandler.GetOverview)
	v1.GET("/statistics/auctions", statisticsHandler.GetAuctionStatistics)
	v1.GET("/statistics/revenue", statisticsHandler.GetRevenueStatistics)
	v1.GET("/statistics/users", statisticsHandler.GetUserStatistics)

	// 类别相关路由
	v1.GET("/categories", categoryHandler.List)
	v1.POST("/categories", categoryHandler.Create)
	v1.PUT("/categories/:id", categoryHandler.Update)
	v1.DELETE("/categories/:id", categoryHandler.Delete)

	// 内部接口（仅服务间调用，不经过 Gateway）
	// spec: docs/superpowers/specs/2026-05-30-h5-missing-c-product-auction.md §5
	// spec: docs/superpowers/specs/2026-05-30-h5-missing-b-livestream.md §4.1
	internalToken := os.Getenv("INTERNAL_API_TOKEN")
	if internalToken == "" {
		log.Println("Warning: INTERNAL_API_TOKEN not set; /internal/* endpoints will reject all calls")
	}
	internal := h.Group("/internal", middleware.InternalAuthMiddleware(internalToken))
	internal.GET("/products", internalHandler.ListByCategory)
	internal.POST("/products/batch", internalHandler.BatchByIDs)
	internal.POST("/live-streams/batch", internalHandler.BatchLiveStreams)
}