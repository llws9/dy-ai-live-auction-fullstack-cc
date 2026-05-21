package router

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// ProductRouter 商品服务路由
type ProductRouter struct {
	client   *client.Client
	baseURL  string
}

// NewProductRouter 创建商品服务路由
func NewProductRouter(productServiceAddr string) *ProductRouter {
	return &ProductRouter{
		client:  client.DefaultClient,
		baseURL: "http://" + productServiceAddr,
	}
}

// RegisterRoutes 注册商品服务路由
func (r *ProductRouter) RegisterRoutes(g *Ctx) {
	// 商品 CRUD
	g.GET("/products", r.proxy)
	g.GET("/products/:id", r.proxy)
	g.POST("/products", r.proxy)
	g.PUT("/products/:id", r.proxy)
	g.DELETE("/products/:id", r.proxy)

	// 竞拍规则
	g.POST("/products/:id/rules", r.proxy)
	g.GET("/products/:id/rules", r.proxy)
}

// proxy 代理请求到商品服务
func (r *ProductRouter) proxy(ctx context.Context, c *app.RequestContext) {
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
