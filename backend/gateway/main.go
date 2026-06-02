package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"gateway-service/config"
	"gateway-service/dao"
	_ "gateway-service/docs" // Swagger docs
	"gateway-service/middleware"
	"gateway-service/pkg/growthbook"
	"gateway-service/pkg/metrics"
	"gateway-service/router"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/hertz-contrib/cors"
)

func main() {
	// 从 Nacos 加载配置（失败时使用环境变量）
	cfg, nacosLoader := config.LoadFromNacosWithFallback()

	// 初始化 Prometheus 指标
	m := metrics.Init("gateway")
	log.Println("Metrics initialized for gateway service")

	// 初始化 GrowthBook A/B 测试客户端 (官方 Go SDK 封装)
	ctx := context.Background()
	gbClient, err := growthbook.NewClient(
		ctx,
		cfg.GrowthBook.APIHost,
		cfg.GrowthBook.ClientKey,
		cfg.GrowthBook.Enabled,
		m,
	)
	if err != nil {
		log.Printf("Warning: GrowthBook client init failed, A/B disabled: %v", err)
	}
	if gbClient != nil {
		defer func() { _ = gbClient.Close() }()
	}
	log.Printf("GrowthBook client initialized (enabled: %v)", cfg.GrowthBook.Enabled)

	// 初始化 Redis 连接
	redisClient, err := dao.InitRedis(cfg.Redis.Addr, cfg.Redis.Password)
	if err != nil {
		log.Printf("Warning: Redis connection failed: %v, rate limiting disabled", err)
	}

	// 监听配置变更（如果 Nacos 可用）
	if nacosLoader != nil {
		go func() {
			// 配置变更时的回调
			_ = nacosLoader.LoadAndListen(cfg, func(newCfg interface{}) {
				log.Printf("Gateway config updated from Nacos")
				// 可以在这里处理配置热更新逻辑
				// 例如更新 Redis 连接、重新初始化 GrowthBook 等
			})
		}()
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
	h.Use(middleware.RequestLogger(middleware.LoggerConfig{
		ServiceName: "gateway-service",
	}))

	// Metrics 中间件
	h.Use(middleware.MetricsMiddleware("gateway", m))

	// 注册路由（包含 GrowthBook 客户端）
	router.RegisterRoutes(h, cfg, gbClient)

	// 前端埋点 API
	h.POST("/api/v1/track", metrics.TrackEvent(m))

	// 启动 Prometheus 指标服务（独立端口 9090）
	go func() {
		promServer := &http.Server{
			Addr:    ":9090",
			Handler: metrics.Handler(),
		}
		log.Printf("Prometheus metrics server starting on :9090")
		if err := promServer.ListenAndServe(); err != nil {
			log.Printf("Prometheus server error: %v", err)
		}
	}()

	// 启动服务
	log.Printf("Gateway service starting on %s", cfg.Server.Port)
	log.Printf("Product Service: %s", cfg.Services.ProductURL)
	log.Printf("Auction Service: %s", cfg.Services.AuctionURL)
	h.Spin()
}
