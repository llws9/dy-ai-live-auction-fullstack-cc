package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"

	"auction-service/config"
	"auction-service/dao"
	"auction-service/handler"
	"auction-service/model"
	"auction-service/mq"
	"auction-service/service"
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

	// 自动迁移表结构（如果表已存在，忽略错误）
	if err := db.AutoMigrate(
		&model.Auction{},
		&model.Bid{},
		&model.UserLiveStreamFollow{},
		&model.UserProductReminder{},
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

	// 设置出价服务的通知发送器
	bidService.SetNotificationSender(notificationService)

	// 初始化 RabbitMQ 连接
	if cfg.RabbitMQ.Host != "" && cfg.RabbitMQ.User != "" {
		rmqConfig := &mq.RabbitMQConfig{
			Host:     cfg.RabbitMQ.Host,
			Port:     fmt.Sprintf("%d", cfg.RabbitMQ.Port),
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
	bidHandler := handler.NewBidHandler(bidService)
	wsHandler := handler.NewWSHandler()
	userHandler := handler.NewUserHandler(userDAO)
	notificationHandler := handler.NewNotificationHandler(notificationService)
	followHandler := handler.NewFollowHandler(followService)

	// 初始化认证 Handler
	jwtExpire := 24 // 24小时
	authHandler := handler.NewAuthHandler(userDAO, cfg.JWT.Secret, jwtExpire)

	// 设置 WebSocket Hub 和 JWT 密钥到 Handler
	wsHandler.SetHub(hub)
	wsHandler.SetJWTSecret(cfg.JWT.Secret)

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

	// 注册路由
	registerRoutes(h, auctionHandler, bidHandler, wsHandler, userHandler, authHandler, notificationHandler, followHandler)

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

// registerRoutes 注册路由
func registerRoutes(h *server.Hertz, auctionHandler *handler.AuctionHandler, bidHandler *handler.BidHandler, wsHandler *handler.WSHandler, userHandler *handler.UserHandler, authHandler *handler.AuthHandler, notificationHandler *handler.NotificationHandler, followHandler *handler.FollowHandler) {
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
	v1.GET("/user/followed-live-streams", followHandler.GetUserFollowsHandler)
	v1.PUT("/live-streams/:id/notification", followHandler.ToggleNotificationHandler)
}