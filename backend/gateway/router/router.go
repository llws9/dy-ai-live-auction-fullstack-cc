package router

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	docs "gateway-service/docs"

	"gateway-service/config"
	"gateway-service/handler"
	"gateway-service/middleware"
)

// RegisterRoutes 注册所有路由
func RegisterRoutes(h *server.Hertz, cfg *config.Config) {
	// 创建代理处理器
	productProxy := handler.NewProxyHandler(cfg.Services.ProductURL)
	auctionProxy := handler.NewProxyHandler(cfg.Services.AuctionURL)

	// ========== Swagger 文档 ==========
	docs.Register(h)

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
	v1.GET("/auctions/:id/ranking", auctionProxy.Forward)

	// ========== WebSocket 路由 ==========
	v1.GET("/ws", func(ctx context.Context, c *app.RequestContext) {
		// WebSocket 连接处理
		handler.HandleWebSocket(ctx, c, cfg.Services.AuctionURL)
	})

	// ========== 订单服务路由 ==========
	v1.GET("/orders", productProxy.Forward)
	v1.GET("/orders/:id", productProxy.Forward)
	v1.POST("/orders/:id/pay", productProxy.Forward)

	// ========== 通知服务路由 ==========
	authGroup.GET("/notifications", auctionProxy.Forward)
	authGroup.GET("/notifications/unread-count", auctionProxy.Forward)
	authGroup.PUT("/notifications/:id/read", auctionProxy.Forward)
	authGroup.PUT("/notifications/read-all", auctionProxy.Forward)

	// ========== 统计服务路由（需要管理员权限） ==========
	authGroup.GET("/statistics/overview", middleware.RequireAdmin(), productProxy.Forward)
	authGroup.GET("/statistics/auctions", middleware.RequireAdmin(), productProxy.Forward)
	authGroup.GET("/statistics/revenue", middleware.RequireAdmin(), productProxy.Forward)
	authGroup.GET("/statistics/users", middleware.RequireAdmin(), productProxy.Forward)
}

// checkService 检查服务是否可用
func checkService(url string) bool {
	// 简单的健康检查实现
	// 实际生产环境应该发送 HTTP 请求检查
	return true
}
