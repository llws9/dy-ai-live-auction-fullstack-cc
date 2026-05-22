package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"

	"auction-service/dao"
	"auction-service/handler"
	"auction-service/model"
	"auction-service/service"
	"auction-service/websocket"
)

func main() {
	// 初始化数据库连接
	db, err := dao.InitDBFromEnv()
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}

	// 初始化 Redis 连接
	_, err = dao.InitRedisFromEnv()
	if err != nil {
		log.Fatalf("Failed to connect redis: %v", err)
	}

	// 自动迁移表结构（如果表已存在，忽略错误）
	if err := db.AutoMigrate(
		&model.Auction{},
		&model.Bid{},
	); err != nil {
		log.Printf("Warning: AutoMigrate failed (tables may already exist): %v", err)
	}

	// 初始化 DAO 层
	auctionDAO := dao.NewAuctionDAO(db)
	bidDAO := dao.NewBidDAO(db)
	ruleDAO := dao.NewAuctionRuleDAO(db)
	userDAO := dao.NewUserDAO(db)
	notificationDAO := dao.NewNotificationDAO(db, dao.GetRedis())

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

	// 设置出价服务的通知发送器
	bidService.SetNotificationSender(notificationService)

	// 初始化 Handler 层
	auctionHandler := handler.NewAuctionHandler(auctionService)
	bidHandler := handler.NewBidHandler(bidService)
	wsHandler := handler.NewWSHandler()
	userHandler := handler.NewUserHandler(userDAO)
	notificationHandler := handler.NewNotificationHandler(notificationService)

	// 初始化认证 Handler
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-in-production" // 默认密钥，生产环境应使用环境变量
	}
	jwtExpire := 24 // 24小时
	authHandler := handler.NewAuthHandler(userDAO, jwtSecret, jwtExpire)

	// 设置 WebSocket Hub 和 JWT 密钥到 Handler
	wsHandler.SetHub(hub)
	wsHandler.SetJWTSecret(jwtSecret)

	// 启动状态转换定时任务
	scheduler := service.NewScheduler(auctionService)
	scheduler.SetHub(hub)
	scheduler.Start()
	defer scheduler.Stop()

	// 启动独立的 WebSocket 服务器（端口 8083）
	go startWebSocketServer(hub, wsHandler)

	// 创建 Hertz 服务器（HTTP，端口 8082）
	h := server.Default(
		server.WithHostPorts(":8082"),
	)

	// 注册路由
	registerRoutes(h, auctionHandler, bidHandler, wsHandler, userHandler, authHandler, notificationHandler)

	// 启动服务
	log.Println("Auction service starting on :8082 (HTTP) and :8083 (WebSocket)")
	h.Spin()
}

// startWebSocketServer 启动独立的 WebSocket 服务器
func startWebSocketServer(hub *websocket.Hub, wsHandler *handler.WSHandler) {
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
		Addr:         ":8083",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Println("WebSocket server starting on :8083")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("WebSocket server error: %v", err)
	}
}

// registerRoutes 注册路由
func registerRoutes(h *server.Hertz, auctionHandler *handler.AuctionHandler, bidHandler *handler.BidHandler, wsHandler *handler.WSHandler, userHandler *handler.UserHandler, authHandler *handler.AuthHandler, notificationHandler *handler.NotificationHandler) {
	v1 := h.Group("/api/v1")

	// ========== 认证相关路由 ==========
	v1.POST("/auth/register", authHandler.Register)
	v1.POST("/auth/login", authHandler.Login)
	v1.GET("/users/me", authHandler.GetCurrentUser)

	// 用户管理相关路由（保留用于测试）
	v1.POST("/users", userHandler.CreateUser)
	v1.POST("/users/batch", userHandler.BatchCreateUsers)

	// 竞拍管理相关路由
	v1.GET("/auctions", auctionHandler.List)        // 获取竞拍列表
	v1.POST("/auctions", auctionHandler.Create)     // 创建竞拍
	v1.GET("/auctions/:id", auctionHandler.Get)     // 获取竞拍详情
	v1.PUT("/auctions/:id/cancel", auctionHandler.Cancel)
	v1.GET("/auctions/:id/result", auctionHandler.GetResult)

	// 出价相关路由
	v1.POST("/auctions/:id/bids", bidHandler.PlaceBid)
	v1.GET("/auctions/:id/ranking", bidHandler.GetRanking)

	// ========== 通知相关路由 ==========
	v1.GET("/notifications", notificationHandler.List)
	v1.GET("/notifications/unread-count", notificationHandler.GetUnreadCount)
	v1.PUT("/notifications/:id/read", notificationHandler.MarkAsRead)
	v1.PUT("/notifications/read-all", notificationHandler.MarkAllAsRead)
}
