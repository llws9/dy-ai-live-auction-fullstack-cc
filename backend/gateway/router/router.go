package router

import (
	"github.com/cloudwego/hertz/pkg/app/server"
)

// RegisterRoutes 注册所有路由
func RegisterRoutes(h *server.Hertz) {
	v1 := h.Group("/api/v1")

	// 商品服务路由
	registerProductRoutes(v1)

	// 竞拍服务路由
	registerAuctionRoutes(v1)

	// 订单服务路由
	registerOrderRoutes(v1)
}

// registerProductRoutes 注册商品相关路由
func registerProductRoutes(g *Ctx) {
	// 商品 CRUD
	g.GET("/products", productHandler.List)
	g.GET("/products/:id", productHandler.Get)
	g.POST("/products", productHandler.Create)
	g.PUT("/products/:id", productHandler.Update)

	// 竞拍规则配置
	g.POST("/products/:id/rules", ruleHandler.Create)
	g.GET("/products/:id/rules", ruleHandler.Get)
}

// registerAuctionRoutes 注册竞拍相关路由
func registerAuctionRoutes(g *Ctx) {
	// 出价
	g.POST("/auctions/:id/bids", bidHandler.PlaceBid)
	g.GET("/auctions/:id/ranking", bidHandler.GetRanking)

	// 竞拍管理
	g.PUT("/auctions/:id/cancel", auctionHandler.Cancel)
	g.GET("/auctions/:id/result", auctionHandler.GetResult)

	// WebSocket 连接
	g.GET("/ws", wsHandler.Handle)
}

// registerOrderRoutes 注册订单相关路由
func registerOrderRoutes(g *Ctx) {
	g.GET("/orders", orderHandler.List)
	g.GET("/orders/:id", orderHandler.Get)
	g.POST("/orders/:id/pay", orderHandler.Pay)
}
