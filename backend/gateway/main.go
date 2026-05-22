package main

import (
	"log"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/hertz-contrib/cors"
	_ "gateway-service/docs" // Swagger docs
	"gateway-service/config"
	"gateway-service/dao"
	"gateway-service/middleware"
	"gateway-service/router"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化 Redis 连接
	redisClient, err := dao.InitRedisFromEnv()
	if err != nil {
		log.Printf("Warning: Redis connection failed: %v, rate limiting disabled", err)
	}

	// 创建 Hertz 服务器
	h := server.Default(
		server.WithHostPorts(cfg.Server.Port),
	)

	// CORS 中间件
	h.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-User-ID"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 全局限流中间件
	if redisClient != nil {
		h.Use(middleware.IPRateLimit(redisClient, cfg.RateLimit.RequestsPerSecond, time.Second))
	}

	// 请求日志中间件
	h.Use(middleware.RequestLogger())

	// JWT 认证中间件（可选，某些路由不需要）
	// h.Use(middleware.JWTAuth(cfg.JWT.Secret))

	// 注册路由
	router.RegisterRoutes(h, cfg)

	// 启动服务
	log.Printf("Gateway service starting on %s", cfg.Server.Port)
	log.Printf("Product Service: %s", cfg.Services.ProductURL)
	log.Printf("Auction Service: %s", cfg.Services.AuctionURL)
	h.Spin()
}
