package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"product-service/client"
	"product-service/config"
	"product-service/dao"
	"product-service/handler"
	"product-service/middleware"
	"product-service/model"
	"product-service/service"
	sharedllm "shared/llm"
)

type categoryNameAdapter struct{ dao *dao.CategoryDAO }

func (a categoryNameAdapter) GetNameByID(ctx context.Context, id int64) (string, bool, error) {
	cat, err := a.dao.GetByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return cat.Name, true, nil
}

func main() {
	// 从 Nacos 加载配置（失败时使用环境变量）
	cfg, nacosLoader := config.LoadFromNacosWithFallback()
	config.ResolveLLMSecrets(cfg)

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
		&model.AuctionRuleTemplate{},
		&model.Order{},
		&model.LiveStream{},
	); err != nil {
		log.Printf("Warning: AutoMigrate failed (tables may already exist): %v", err)
	}
	if err := dao.EnsureAuctionRuleProductScopeSchema(db); err != nil {
		log.Fatalf("Failed to align auction_rules schema: %v", err)
	}

	// 初始化 DAO 层
	productDAO := dao.NewProductDAO(db)
	ruleDAO := dao.NewAuctionRuleDAO(db)
	ruleTemplateDAO := dao.NewAuctionRuleTemplateDAO(db)
	orderDAO := dao.NewOrderDAO(db)
	historyDAO := dao.NewHistoryDAO(db)
	orderAdminDAO := dao.NewOrderAdminDAO(db)
	statisticsDAO := dao.NewStatisticsDAO(db)
	liveStreamDAO := dao.NewLiveStreamDAO(db)
	categoryDAO := dao.NewCategoryDAO(db)

	// 初始化 Service 层
	productService := service.NewProductService(productDAO, ruleDAO, liveStreamDAO)
	ruleTemplateService := service.NewAuctionRuleTemplateService(ruleTemplateDAO)
	orderService := service.NewOrderService(orderDAO, historyDAO)
	orderService.SetAdminOrderDAO(orderAdminDAO)
	orderService.SetProductDAO(productDAO)
	statisticsService := service.NewStatisticsService(statisticsDAO)
	var viewerCounter service.LiveViewerCounter = service.ZeroLiveViewerCounter{}
	var redisClient *redis.Client
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     redisAddr,
			Password: cfg.Redis.Password,
			PoolSize: cfg.Redis.PoolSize,
		})
		viewerCounter = service.NewRedisLiveViewerCounter(redisClient)
	}
	liveStreamService := service.NewLiveStreamServiceWithMetrics(liveStreamDAO, viewerCounter)
	categoryService := service.NewCategoryService(categoryDAO)
	log.Printf("LLM provider configured provider=%q base_url=%q model=%q timeout_ms=%d api_key_set=%t",
		cfg.LLM.Provider,
		cfg.LLM.Doubao.BaseURL,
		cfg.LLM.Doubao.Model,
		cfg.LLM.TimeoutMs,
		cfg.LLM.Doubao.APIKey != "",
	)
	llmProvider := sharedllm.NewDoubaoProvider(sharedllm.DoubaoOptions{
		BaseURL: cfg.LLM.Doubao.BaseURL,
		APIKey:  cfg.LLM.Doubao.APIKey,
		Model:   cfg.LLM.Doubao.Model,
		Timeout: time.Duration(cfg.LLM.TimeoutMs) * time.Millisecond,
	})
	copyService := service.NewCopywritingService(llmProvider, categoryNameAdapter{dao: categoryDAO}, redisClient, cfg.LLM.Doubao.Model)

	// 初始化 Handler 层
	productHandler := handler.NewProductHandler(productService)
	ruleHandler := handler.NewRuleHandler(productService)
	ruleTemplateHandler := handler.NewAuctionRuleTemplateHandler(ruleTemplateService)
	orderHandler := handler.NewOrderHandler(orderService)
	statisticsHandler := handler.NewStatisticsHandler(statisticsService)
	productPublishHandler := handler.NewProductHandler(productService)
	liveStreamHandler := handler.NewLiveStreamHandler(liveStreamService)
	auctionSvcURL := os.Getenv("AUCTION_SERVICE_URL")
	if auctionSvcURL == "" {
		auctionSvcURL = "http://localhost:8082"
	}
	auctionClient := client.NewAuctionClient(auctionSvcURL, 2*time.Second)
	auctionClient.SetInternalToken(os.Getenv("INTERNAL_API_TOKEN"))
	liveStreamHandler.SetAuctionClient(auctionClient)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	copywritingHandler := handler.NewCopywritingHandler(copyService)
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
	httpMetrics := middleware.NewHTTPMetrics("product", prometheus.DefaultRegisterer)
	h.Use(middleware.MetricsMiddleware("product", httpMetrics))

	// 注册路由
	registerRoutes(h, productHandler, ruleHandler, ruleTemplateHandler, orderHandler, statisticsHandler, productPublishHandler, liveStreamHandler, categoryHandler, copywritingHandler, internalHandler)
	h.GET("/metrics", func(ctx context.Context, c *app.RequestContext) {
		middleware.WriteMetricsResponse(c, prometheus.DefaultGatherer)
	})

	// 启动服务
	log.Printf("Product service starting on %s", cfg.Server.Port)
	h.Spin()
}

// registerRoutes 注册路由
func registerRoutes(h *server.Hertz, productHandler *handler.ProductHandler, ruleHandler *handler.RuleHandler, ruleTemplateHandler *handler.AuctionRuleTemplateHandler, orderHandler *handler.OrderHandler, statisticsHandler *handler.StatisticsHandler, productPublishHandler *handler.ProductHandler, liveStreamHandler *handler.LiveStreamHandler, categoryHandler *handler.CategoryHandler, copywritingHandler *handler.CopywritingHandler, internalHandler *handler.InternalHandler) {
	v1 := h.Group("/api/v1")

	// 商品相关路由
	v1.GET("/products", productHandler.List)
	v1.GET("/products/:id", productHandler.Get)
	v1.POST("/products", productHandler.Create)
	v1.PUT("/products/:id", productHandler.Update)
	v1.DELETE("/products/:id", productHandler.Delete)

	// AI 文案生成（商家/管理员）
	v1.POST("/products/ai/copywriting", copywritingHandler.Generate)

	// 商品发布/下架路由
	v1.POST("/products/:id/publish", productPublishHandler.PublishHandler)
	v1.POST("/products/:id/unpublish", productPublishHandler.UnpublishHandler)

	// 竞拍规则相关路由
	v1.POST("/products/:id/rules", ruleHandler.Create)
	v1.GET("/products/:id/rules", ruleHandler.Get)

	internalToken := os.Getenv("INTERNAL_API_TOKEN")
	if internalToken == "" {
		log.Println("Warning: INTERNAL_API_TOKEN not set; protected internal endpoints will reject all calls")
	}
	internalAuth := middleware.InternalAuthMiddleware(internalToken)

	// 订单相关路由
	v1.GET("/orders", orderHandler.List)
	v1.GET("/orders/summary", orderHandler.Summary)
	v1.GET("/orders/:id", orderHandler.Get)
	v1.PUT("/orders/:id", orderHandler.Update)
	v1.POST("/orders/:id/pay", orderHandler.Pay)
	v1.PUT("/orders/:id/ship", orderHandler.Ship)
	v1.GET("/orders/history", orderHandler.GetUserHistory)

	// 订单 admin 路由：必须经 Gateway 透传内部 token，且下游二次校验 X-User-Role=admin。
	v1.GET("/admin/products", internalAuth, productHandler.AdminList)
	v1.GET("/admin/products/:id", internalAuth, productHandler.AdminGet)
	v1.POST("/admin/products", internalAuth, productHandler.AdminCreate)
	v1.PUT("/admin/products/:id", internalAuth, productHandler.AdminUpdate)
	v1.DELETE("/admin/products/:id", internalAuth, productHandler.AdminDelete)
	v1.GET("/admin/auction-rule-templates", internalAuth, ruleTemplateHandler.List)
	v1.GET("/admin/auction-rule-templates/:id", internalAuth, ruleTemplateHandler.Get)
	v1.POST("/admin/auction-rule-templates", internalAuth, ruleTemplateHandler.Create)
	v1.PUT("/admin/auction-rule-templates/:id", internalAuth, ruleTemplateHandler.Update)
	v1.DELETE("/admin/auction-rule-templates/:id", internalAuth, ruleTemplateHandler.Delete)
	v1.GET("/admin/orders", internalAuth, orderHandler.AdminList)
	v1.GET("/admin/orders/:id", internalAuth, orderHandler.AdminGet)

	// 直播间 admin 路由：必须经 Gateway 透传内部 token，且由 Gateway 校验管理员身份。
	v1.GET("/admin/live-streams", internalAuth, liveStreamHandler.ListAdmin)
	v1.GET("/admin/live-streams/:id", internalAuth, liveStreamHandler.AdminGet)
	v1.POST("/admin/live-streams", internalAuth, liveStreamHandler.AdminCreate)
	v1.PUT("/admin/live-streams/:id", internalAuth, liveStreamHandler.AdminUpdate)
	v1.PUT("/admin/live-streams/:id/end", internalAuth, liveStreamHandler.EndAdmin)
	v1.PUT("/admin/live-streams/:id/ban", internalAuth, liveStreamHandler.BanAdmin)
	v1.GET("/live-streams", liveStreamHandler.ListPublic)
	v1.GET("/live-streams/:id", liveStreamHandler.GetDetail)

	// 统计相关路由：经 Gateway 注入 internal token，下游再按 X-User-Role/X-User-ID 做范围校验。
	v1.GET("/statistics/overview", internalAuth, statisticsHandler.GetOverview)
	v1.GET("/statistics/auctions", internalAuth, statisticsHandler.GetAuctionStatistics)
	v1.GET("/statistics/revenue", internalAuth, statisticsHandler.GetRevenueStatistics)
	v1.GET("/statistics/users", internalAuth, statisticsHandler.GetUserStatistics)

	// 类别相关路由
	v1.GET("/categories", categoryHandler.List)
	v1.POST("/categories", categoryHandler.Create)
	v1.PUT("/categories/:id", categoryHandler.Update)
	v1.DELETE("/categories/:id", categoryHandler.Delete)

	// 内部接口（仅服务间调用，不经过 Gateway）
	// spec: docs/superpowers/specs/2026-05-30-h5-missing-c-product-auction.md §5
	// spec: docs/superpowers/specs/2026-05-30-h5-missing-b-livestream.md §4.1
	internal := h.Group("/internal", internalAuth)
	internal.GET("/products", internalHandler.ListByCategory)
	internal.POST("/products/batch", internalHandler.BatchByIDs)
	internal.POST("/live-streams/batch", internalHandler.BatchLiveStreams)
}
