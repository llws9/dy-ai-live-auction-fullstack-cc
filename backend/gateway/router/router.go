package router

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"

	"gateway-service/config"
	"gateway-service/handler"
)

// RegisterRoutes 注册所有路由
func RegisterRoutes(h *server.Hertz, cfg *config.Config) {
	// 创建代理处理器
	productProxy := handler.NewProxyHandler(cfg.Services.ProductURL)
	auctionProxy := handler.NewProxyHandler(cfg.Services.AuctionURL)

	v1 := h.Group("/api/v1")

	// ========== 健康检查与监控 ==========
	h.GET("/health", handler.Health("gateway"))
	h.GET("/ready", handler.Ready("gateway", map[string]func() bool{
		"product_service": func() bool { return checkService(cfg.Services.ProductURL) },
		"auction_service": func() bool { return checkService(cfg.Services.AuctionURL) },
	}))
	h.GET("/metrics", handler.Metrics("gateway"))

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
	v1.POST("/auctions", auctionProxy.Forward)
	v1.PUT("/auctions/:id/cancel", auctionProxy.Forward)
	v1.GET("/auctions/:id/result", auctionProxy.Forward)
	v1.POST("/auctions/:id/bids", auctionProxy.Forward)
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
}

// checkService 检查服务是否可用
func checkService(url string) bool {
	// 简单的健康检查实现
	// 实际生产环境应该发送 HTTP 请求检查
	return true
}
