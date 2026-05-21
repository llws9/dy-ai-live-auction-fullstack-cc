package router

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// AuctionRouter 竞拍服务路由
type AuctionRouter struct {
	client  *client.Client
	baseURL string
}

// NewAuctionRouter 创建竞拍服务路由
func NewAuctionRouter(auctionServiceAddr string) *AuctionRouter {
	return &AuctionRouter{
		client:  client.DefaultClient,
		baseURL: "http://" + auctionServiceAddr,
	}
}

// RegisterRoutes 注册竞拍服务路由
func (r *AuctionRouter) RegisterRoutes(g *Ctx) {
	// 出价
	g.POST("/auctions/:id/bids", r.proxy)
	g.GET("/auctions/:id/ranking", r.proxy)

	// 竞拍管理
	g.PUT("/auctions/:id/cancel", r.proxy)
	g.GET("/auctions/:id/result", r.proxy)
}

// proxy 代理请求到竞拍服务
func (r *AuctionRouter) proxy(ctx context.Context, c *app.RequestContext) {
	// 构建目标 URL
	path := string(c.URI().Path())
	if string(c.URI().QueryString()) != "" {
		path += "?" + string(c.URI().QueryString())
	}
	targetURL := r.baseURL + path

	// 创建代理请求
	req := c.GetRequest()
	req.SetRequestURI(targetURL)
	req.SetMethod(string(c.Method()))

	// 复制请求头
	c.Request.Header.VisitAll(func(key, value []byte) {
		req.Header.SetBytesKV(key, value)
	})

	// 发送请求
	resp, err := r.client.Do(ctx, req)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]interface{}{
			"code":    500,
			"message": "服务不可用: " + err.Error(),
		})
		return
	}

	// 复制响应
	c.Status(resp.StatusCode())
	c.Response.Header.Set("Content-Type", string(resp.Header.Get("Content-Type")))
	c.Response.SetBody(resp.Body())
}
