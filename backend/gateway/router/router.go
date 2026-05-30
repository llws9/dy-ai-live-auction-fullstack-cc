package router

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"

	"gateway-service/config"
	"gateway-service/handler"
	"gateway-service/middleware"
	"gateway-service/pkg/growthbook"
)

// RouterConfig 路由配置
type RouterConfig struct {
	Config    *config.Config
	GBClient  *growthbook.Client
}

// RegisterRoutes 注册所有路由
func RegisterRoutes(h *server.Hertz, cfg *config.Config, gbClient *growthbook.Client) {
	// 创建代理处理器
	productProxy := handler.NewProxyHandler(cfg.Services.ProductURL)
	auctionProxy := handler.NewProxyHandler(cfg.Services.AuctionURL)
	testProxy := handler.NewProxyHandler(cfg.Services.TestURL)

	// 创建实验处理器
	experimentHandler := handler.NewExperimentHandler(gbClient)

	// ========== Swagger 文档 ==========
	// docs.Register(h) // 暂时注释掉swagger

	v1 := h.Group("/api/v1")

	// ========== 健康检查与监控 ==========
	h.GET("/health", handler.Health("gateway"))
	h.GET("/ready", handler.Ready("gateway", map[string]func() bool{
		"product_service": func() bool { return checkService(cfg.Services.ProductURL) },
		"auction_service": func() bool { return checkService(cfg.Services.AuctionURL) },
	}))
	h.GET("/metrics", handler.Metrics("gateway"))

	// ========== 认证路由（无需JWT） ==========
	v1.POST("/auth/register", auctionProxy.Forward)
	v1.POST("/auth/login", auctionProxy.Forward)

	// ========== 需要JWT认证的路由 ==========
	authGroup := v1.Group("")
	authGroup.Use(middleware.JWTAuth(cfg.JWT.Secret))
	authGroup.Use(middleware.ExperimentMiddleware(gbClient))  // 实验中间件

	// 用户信息
	authGroup.GET("/users/me", auctionProxy.Forward)

	// ========== 商品服务路由 ==========
	v1.GET("/products", productProxy.Forward)
	v1.GET("/products/:id", productProxy.Forward)
	v1.POST("/products", productProxy.Forward)
	v1.PUT("/products/:id", productProxy.Forward)
	v1.DELETE("/products/:id", productProxy.Forward)
	v1.POST("/products/:id/rules", productProxy.Forward)
	v1.GET("/products/:id/rules", productProxy.Forward)

	// 商品发布/下架需要商家或管理员权限
	authGroup.POST("/products/:id/publish", middleware.RequireMerchant(), productProxy.Forward)
	authGroup.POST("/products/:id/unpublish", middleware.RequireMerchant(), productProxy.Forward)

	// ========== 竞拍服务路由 ==========
	v1.GET("/auctions", auctionProxy.Forward)
	v1.GET("/auctions/:id", auctionProxy.Forward)
	// 创建竞拍需要主播或管理员权限
	authGroup.POST("/auctions", middleware.RequireStreamer(), auctionProxy.Forward)
	// 取消竞拍需要主播或管理员权限
	authGroup.PUT("/auctions/:id/cancel", middleware.RequireStreamer(), auctionProxy.Forward)
	v1.GET("/auctions/:id/result", auctionProxy.Forward)

	// 出价需要认证
	authGroup.POST("/auctions/:id/bids", auctionProxy.Forward)
	v1.GET("/auctions/:id/bids", auctionProxy.Forward)
	v1.GET("/auctions/:id/ranking", auctionProxy.Forward)

	// ========== 直播间关注路由 ==========
	authGroup.POST("/live-streams/:id/follow", auctionProxy.Forward)
	authGroup.DELETE("/live-streams/:id/follow", auctionProxy.Forward)
	authGroup.GET("/live-streams/:id/follow-status", auctionProxy.Forward) // T2.6 (F-B2)
	authGroup.GET("/user/followed-live-streams", auctionProxy.Forward)
	authGroup.PUT("/live-streams/:id/notification", auctionProxy.Forward)

	// ========== WebSocket 路由 ==========
	v1.GET("/ws", func(ctx context.Context, c *app.RequestContext) {
		// WebSocket 连接处理
		handler.HandleWebSocket(ctx, c, cfg.Services.AuctionURL)
	})

	// ========== 订单服务路由 ==========
	v1.GET("/orders", productProxy.Forward)
	v1.GET("/orders/:id", productProxy.Forward)
	v1.POST("/orders/:id/pay", productProxy.Forward)
	v1.PUT("/orders/:id/ship", productProxy.Forward)          // T007: 订单发货
	// T008: 用户订单历史 — JWT 化（spec C / F-C3, M1 P0 安全修复）
	// 仅认证用户可读；下游通过 X-User-ID header 透传识别本人，禁止接受 query user_id。
	authGroup.GET("/orders/history", productProxy.Forward)

	// ========== 直播间路由 ==========
	authGroup.GET("/admin/live-streams", middleware.RequireAdmin(), productProxy.Forward) // T009: 管理端直播间列表
	// T010: 直播间详情。公开访问，但若客户端带合法 Bearer token，
	// OptionalJWTAuth 会注入 user_id，proxy.Forward 据此把 X-User-ID 透传给 product-service，
	// 用于查询 is_following 等登录态字段（spec B / F-B1, T2.5）。
	v1.GET("/live-streams/:id", middleware.OptionalJWTAuth(cfg.JWT.Secret), productProxy.Forward)

	// ========== 通知服务路由 ==========
	authGroup.GET("/notifications", auctionProxy.Forward)
	authGroup.GET("/notifications/unread-count", auctionProxy.Forward)
	authGroup.PUT("/notifications/:id/read", auctionProxy.Forward)
	authGroup.PUT("/notifications/read-all", auctionProxy.Forward)
	authGroup.POST("/notifications/hot-pull", auctionProxy.Forward)

	// ========== 点天灯订阅路由 ==========
	authGroup.POST("/sky-lamp/subscriptions", auctionProxy.Forward)
	authGroup.PUT("/sky-lamp/subscriptions/:id/stop", auctionProxy.Forward)
	authGroup.GET("/sky-lamp/subscriptions", auctionProxy.Forward)
	authGroup.GET("/sky-lamp/subscriptions/:id", auctionProxy.Forward)

	// ========== 统计服务路由（需要管理员权限） ==========
	authGroup.GET("/statistics/overview", middleware.RequireAdmin(), productProxy.Forward)
	authGroup.GET("/statistics/auctions", middleware.RequireAdmin(), productProxy.Forward)
	authGroup.GET("/statistics/revenue", middleware.RequireAdmin(), productProxy.Forward)
	authGroup.GET("/statistics/users", middleware.RequireAdmin(), productProxy.Forward)

	// ========== 类别服务路由 ==========
	v1.GET("/categories", productProxy.Forward)                                                // T018: 类别列表（公开）
	authGroup.POST("/categories", middleware.RequireAdmin(), productProxy.Forward)            // T019: 类别创建（管理员）
	authGroup.PUT("/categories/:id", middleware.RequireAdmin(), productProxy.Forward)         // T020: 类别更新（管理员）
	authGroup.DELETE("/categories/:id", middleware.RequireAdmin(), productProxy.Forward)      // T021: 类别删除（管理员）

	// ========== A/B 测试实验路由 ==========
	authGroup.GET("/experiments/features", experimentHandler.GetFeatures)       // 获取用户特性开关
	authGroup.POST("/experiments/viewed", experimentHandler.TrackViewed)        // 记录实验查看
	authGroup.POST("/experiments/completed", experimentHandler.TrackCompleted)  // 记录实验完成

	// ========== 测试平台路由（test-service） ==========
	// HTTP 透传 /api/test/* → test-service:18090
	testHTTP := h.Group("/api/test")
	testHTTP.Any("/*path", testProxy.Forward)
	// WS endpoint discovery：/ws/test/progress 返回真实 WS URL
	h.GET("/ws/test/progress", func(ctx context.Context, c *app.RequestContext) {
		handler.HandleTestWebSocket(ctx, c, cfg.Services.TestWSURL)
	})
}

// checkService 检查服务是否可用
func checkService(url string) bool {
	// 简单的健康检查实现
	// 实际生产环境应该发送 HTTP 请求检查
	return true
}
