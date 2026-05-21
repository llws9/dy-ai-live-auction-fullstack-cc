// backend/gateway/main.go

package main

import (
	"log"

	"github.com/cloudwego/hertz/pkg/app/server"

	"gateway-service/dao"
	"gateway-service/middleware"
	"gateway-service/router"
)

func main() {
	// 初始化 Redis 连接
	redisClient, err := dao.InitRedisFromEnv()
	if err != nil {
		log.Fatalf("Failed to connect redis: %v", err)
	}

	// 创建 Hertz 服务器
	h := server.Default(
		server.WithHostPorts(":8080"),
	)

	// 注册全局中间件
	h.Use(middleware.IPRateLimit(redisClient, 1000, time.Second)) // 全局限流：1000 QPS

	// 注册路由
	productRouter := router.NewProductRouter("product:8081")
	auctionRouter := router.NewAuctionRouter("auction:8082")

	v1 := h.Group("/api/v1")
	productRouter.RegisterRoutes(v1)
	auctionRouter.RegisterRoutes(v1)

	// 启动服务
	log.Println("Gateway service starting on :8080")
	h.Spin()
}
