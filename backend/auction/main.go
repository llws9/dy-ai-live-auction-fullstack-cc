package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/prometheus/client_golang/prometheus"

	"auction-service/client"
	"auction-service/config"
	"auction-service/dao"
	"auction-service/handler"
	"auction-service/model"
	"auction-service/mq"
	"auction-service/pkg/metrics"
	"auction-service/service"
	"auction-service/service/cron"
	"auction-service/websocket"
)

func main() {
	// 从 Nacos 加载配置（失败时使用环境变量）
	cfg, nacosLoader := config.LoadFromNacosWithFallback()

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

	// 初始化 Redis 连接
	_, err = dao.InitRedis(cfg.Redis.Addr, cfg.Redis.Password)
	if err != nil {
		log.Fatalf("Failed to connect redis: %v", err)
	}

	// 初始化 Prometheus 指标收集器（统一注册）
	metrics.InitRegistry()
	log.Println("Prometheus metrics initialized successfully")

	// 自动迁移表结构（如果表已存在，忽略错误）
	if err := db.AutoMigrate(
		&model.Auction{},
		&model.Bid{},
		&model.Notification{},
		&model.UserLiveStreamFollow{},
		&model.UserProductReminder{},
		&model.SkyLampSubscription{},
		&model.UserBalance{},
	); err != nil {
		log.Printf("Warning: AutoMigrate failed (tables may already exist): %v", err)
	}

	// 初始化 DAO 层
	auctionDAO := dao.NewAuctionDAO(db)
	bidDAO := dao.NewBidDAO(db)
	ruleDAO := dao.NewAuctionRuleDAO(db)
	userDAO := dao.NewUserDAO(db)
	notificationDAO := dao.NewNotificationDAO(db, dao.GetRedis())
	userLiveStreamFollowDAO := dao.NewUserLiveStreamFollowDAO(db)
	userProductReminderDAO := dao.NewUserProductReminderDAO(db)
	skyLampDAO := dao.NewSkyLampDAO(db)
	userBalanceDAO := dao.NewUserBalanceDAO(db)

	// 初始化 WebSocket Hub
	hub := websocket.NewHub()

	// 创建 WebSocketManager 统一管理 Hub 和 StateManager
	wsManager := websocket.NewWebSocketManager(hub, dao.GetRedis())
	wsManager.Run()
	defer wsManager.Stop()

	// 初始化 Service 层
	auctionService := service.NewAuctionService(auctionDAO)
	bidService := service.NewBidService(auctionDAO, bidDAO, ruleDAO, userDAO)
	notificationService := service.NewNotificationService(notificationDAO, dao.GetRedis())
	batchNotificationService := service.NewBatchNotificationService(userLiveStreamFollowDAO, notificationDAO, notificationService)
	followService := service.NewFollowService(userLiveStreamFollowDAO)
	productReminderService := service.NewProductReminderService(userProductReminderDAO)
	productReminderService.SetAuctionDAO(auctionDAO)

	// 初始化分布式锁服务
	distributedLockService := service.NewDistributedLockService(dao.GetRedis())

	skyLampService := service.NewSkyLampService(skyLampDAO, bidService, cfg.SkyLamp, distributedLockService)

	// 设置出价服务的通知发送器和指标收集器
	bidService.SetNotificationSender(notificationService)
	bidService.SetSkyLampTrigger(skyLampService)
	bidService.SetMetrics(metrics.GetMetrics())
	bidService.SetHub(hub)
	auctionService.SetBidDAO(bidDAO)
	auctionService.SetNotificationSender(notificationService)
	auctionService.SetSkyLampDAO(skyLampDAO)

	// 通知服务指标暂未实现，跳过设置
	notificationService.SetHub(hub)

	// 初始化 RabbitMQ 连接
	if cfg.RabbitMQ.Host != "" && cfg.RabbitMQ.User != "" {
		rmqConfig := &mq.RabbitMQConfig{
			Host:     cfg.RabbitMQ.Host,
			Port:     cfg.RabbitMQ.Port,
			User:     cfg.RabbitMQ.User,
			Password: cfg.RabbitMQ.Password,
			VHost:    cfg.RabbitMQ.VHost,
		}

		rmq, err := mq.NewRabbitMQConnection(rmqConfig)
		if err != nil {
			log.Printf("Warning: RabbitMQ connection failed: %v, notification queue disabled", err)
		} else {
			defer rmq.Close()

			// 初始化通知处理器和消费者
			notifyHandler := mq.NewNotificationHandler(batchNotificationService)
			consumer := mq.NewNotificationConsumer(rmq, notifyHandler)

			// 启动消费者
			if err := consumer.Start(); err != nil {
				log.Printf("Warning: Failed to start RabbitMQ consumer: %v", err)
			} else {
				log.Println("RabbitMQ consumer started successfully")
			}
		}
	} else {
		log.Printf("Warning: RabbitMQ config incomplete, notification queue disabled")
	}

	// 初始化 Handler 层
	auctionHandler := handler.NewAuctionHandler(auctionService)
	// 注入 product-service 内部接口客户端，启用 list 接口的 category 过滤与商品摘要回填（spec C §5.2）。
	productSvcURL := os.Getenv("PRODUCT_SERVICE_URL")
	if productSvcURL == "" {
		productSvcURL = "http://localhost:8081"
	}
	auctionHandler.SetProductClient(client.NewHTTPProductClient(productSvcURL, 2*time.Second))
	bidHandler := handler.NewBidHandler(bidService)
	wsHandler := handler.NewWSHandler()
	userHandler := handler.NewUserHandler(userDAO)
	notificationHandler := handler.NewNotificationHandler(notificationService)
	followHandler := handler.NewFollowHandler(followService)
	productReminderHandler := handler.NewProductReminderHandler(productReminderService)
	skyLampHandler := handler.NewSkyLampHandler(skyLampService)
	userBalanceHandler := handler.NewUserBalanceHandler(userBalanceDAO)

	// 初始化认证 Handler
	jwtExpire := 24 // 24小时
	authHandler := handler.NewAuthHandler(userDAO, cfg.JWT.Secret, jwtExpire)

	// 设置 WebSocket Hub 和 JWT 密钥到 Handler
	wsHandler.SetHub(hub)
	wsHandler.SetJWTSecret(cfg.JWT.Secret)

	// 设置 WebSocket Hub 到 NotificationService（用于实时推送）
	notificationService.SetHub(hub)
	notificationService.SetFollowDAO(userLiveStreamFollowDAO) // 用于热拉Redis失败时DB兜底

	// 启动状态转换定时任务
	scheduler := service.NewScheduler(auctionService)
	scheduler.SetHub(hub)
	scheduler.Start()
	defer scheduler.Stop()

	// 启动冷推定时任务
	ctx := context.Background()
	coldPushScheduler := service.NewColdPushScheduler(notificationService, userLiveStreamFollowDAO, dao.GetRedis())
	go coldPushScheduler.Run(ctx)
	log.Println("Cold push scheduler started")

	// 启动热度自动更新定时任务
	liveStreamStatsService := service.NewLiveStreamStatsService()
	statsCron := cron.NewStatsCron(userLiveStreamFollowDAO, liveStreamStatsService)
	statsCron.Start(ctx)
	defer statsCron.Stop()
	log.Println("Stats cron started for auto-updating hotness")

	// 监听配置变更（如果 Nacos 可用）
	if nacosLoader != nil {
		go func() {
			_ = nacosLoader.LoadAndListen(cfg, func(newCfg interface{}) {
				log.Printf("Auction config updated from Nacos")
			})
		}()
	}

	// 启动独立的 WebSocket 服务器
	go startWebSocketServer(hub, wsHandler, cfg.Server.WSPort)

	// 创建 Hertz 服务器（HTTP）
	h := server.Default(
		server.WithHostPorts(cfg.Server.HTTPPort),
	)
	h.Use(gatewayIdentityMiddleware())

	// 注册路由
	registerRoutes(h, auctionHandler, bidHandler, wsHandler, userHandler, authHandler, notificationHandler, followHandler, productReminderHandler, skyLampHandler, userBalanceHandler)

	// 注册 Prometheus metrics 端点
	h.GET("/metrics", func(ctx context.Context, c *app.RequestContext) {
		c.Response.Header.Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		// 直接采集 metrics
		mfs, err := prometheus.DefaultGatherer.Gather()
		if err != nil {
			c.Response.Header.Set("Content-Type", "text/plain")
			c.Response.SetBody([]byte(fmt.Sprintf("Error gathering metrics: %v", err)))
			return
		}
		// 将 metrics 写入响应
		for _, mf := range mfs {
			c.Response.AppendBody([]byte(fmt.Sprintf("%s\n", mf.String())))
		}
	})

	// 启动服务
	log.Printf("Auction service starting on %s (HTTP) and %s (WebSocket)", cfg.Server.HTTPPort, cfg.Server.WSPort)
	h.Spin()
}

// startWebSocketServer 启动独立的 WebSocket 服务器
func startWebSocketServer(hub *websocket.Hub, wsHandler *handler.WSHandler, port string) {
	// 使用标准库的 ServeMux
	mux := http.NewServeMux()

	// WebSocket 路由
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// 获取 auction_id 参数
		auctionIDStr := r.URL.Query().Get("auction_id")
		if auctionIDStr == "" {
			http.Error(w, "auction_id is required", http.StatusBadRequest)
			return
		}

		auctionID, err := strconv.ParseInt(auctionIDStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid auction_id", http.StatusBadRequest)
			return
		}

		// 处理 WebSocket 连接
		wsHandler.HandleWebSocket(hub, auctionID, w, r)
	})

	// 健康检查
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"websocket"}`))
	})

	server := &http.Server{
		Addr:         port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("WebSocket server starting on %s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("WebSocket server error: %v", err)
	}
}

func gatewayIdentityMiddleware() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if userIDHeader := string(c.GetHeader("X-User-ID")); userIDHeader != "" {
			if userID, err := strconv.ParseInt(userIDHeader, 10, 64); err == nil {
				c.Set("user_id", userID)
			}
		}

		if username := string(c.GetHeader("X-Username")); username != "" {
			c.Set("username", username)
		}

		if role := parseGatewayRole(string(c.GetHeader("X-User-Role"))); role >= 0 {
			c.Set("user_role", role)
		}

		c.Next(ctx)
	}
}

func parseGatewayRole(role string) int {
	switch role {
	case "admin":
		return 2
	case "streamer", "merchant":
		return 1
	case "user":
		return 0
	default:
		return -1
	}
}

// registerRoutes 注册路由
func registerRoutes(h *server.Hertz, auctionHandler *handler.AuctionHandler, bidHandler *handler.BidHandler, wsHandler *handler.WSHandler, userHandler *handler.UserHandler, authHandler *handler.AuthHandler, notificationHandler *handler.NotificationHandler, followHandler *handler.FollowHandler, productReminderHandler *handler.ProductReminderHandler, skyLampHandler *handler.SkyLampHandler, userBalanceHandler *handler.UserBalanceHandler) {
	v1 := h.Group("/api/v1")

	// ========== 认证相关路由 ==========
	v1.POST("/auth/register", authHandler.Register)
	v1.POST("/auth/login", authHandler.Login)
	v1.GET("/users/me", authHandler.GetCurrentUser)

	// 用户管理相关路由（保留用于测试）
	v1.POST("/users", userHandler.CreateUser)
	v1.POST("/users/batch", userHandler.BatchCreateUsers)

	// 竞拍管理相关路由
	v1.GET("/auctions", auctionHandler.List)
	v1.POST("/auctions", auctionHandler.Create)
	v1.GET("/auctions/:id", auctionHandler.Get)
	v1.PUT("/auctions/:id/cancel", auctionHandler.Cancel)
	v1.GET("/auctions/:id/result", auctionHandler.GetResult)

	// 出价相关路由
	v1.POST("/auctions/:id/bids", bidHandler.PlaceBid)
	v1.GET("/auctions/:id/bids", auctionHandler.GetBids)
	v1.GET("/auctions/:id/ranking", bidHandler.GetRanking)

	// ========== 通知相关路由 ==========
	v1.GET("/notifications", notificationHandler.List)
	v1.GET("/notifications/unread-count", notificationHandler.GetUnreadCount)
	v1.PUT("/notifications/:id/read", notificationHandler.MarkAsRead)
	v1.PUT("/notifications/read-all", notificationHandler.MarkAllAsRead)
	v1.POST("/notifications/hot-pull", notificationHandler.HotPullNotifications)

	// ========== 直播间关注相关路由 ==========
	v1.POST("/live-streams/:id/follow", followHandler.FollowHandler)
	v1.DELETE("/live-streams/:id/follow", followHandler.UnfollowHandler)
	v1.GET("/live-streams/:id/follow-status", followHandler.GetFollowStatusHandler) // T2.6 (F-B2)
	v1.GET("/user/followed-live-streams", followHandler.GetUserFollowsHandler)
	v1.PUT("/live-streams/:id/notification", followHandler.ToggleNotificationHandler)

	// ========== 商品提醒订阅相关路由 ==========
	v1.POST("/products/:id/remind", productReminderHandler.SubscribeProductReminder)
	v1.DELETE("/products/:id/remind", productReminderHandler.UnsubscribeProductReminder)
	v1.GET("/users/me/reminders", productReminderHandler.GetUserReminders)

	// ========== 点天灯订阅相关路由 ==========
	v1.POST("/sky-lamp/subscriptions", skyLampHandler.StartSubscription)
	v1.PUT("/sky-lamp/subscriptions/:id/stop", skyLampHandler.StopSubscription)
	v1.GET("/sky-lamp/subscriptions", skyLampHandler.GetUserSubscriptions)
	v1.GET("/sky-lamp/subscriptions/:id", skyLampHandler.GetSubscriptionDetail)

	// ========== 用户余额（T3.1 F-A2 只读） ==========
	v1.GET("/user/balance", userBalanceHandler.GetUserBalanceHandler)
}
