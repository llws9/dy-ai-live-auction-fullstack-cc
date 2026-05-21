package main

import (
	"log"

	"github.com/cloudwego/hertz/pkg/app/server"

	"auction-service/dao"
	"auction-service/handler"
	"auction-service/model"
	"auction-service/service"
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

	// 初始化 Service 层
	auctionService := service.NewAuctionService(auctionDAO)
	bidService := service.NewBidService(auctionDAO, bidDAO, ruleDAO)

	// 初始化 Handler 层
	auctionHandler := handler.NewAuctionHandler(auctionService)
	bidHandler := handler.NewBidHandler(bidService)

	// 启动状态转换定时任务
	scheduler := service.NewScheduler(auctionService)
	scheduler.Start()
	defer scheduler.Stop()

	// 创建 Hertz 服务器（HTTP）
	h := server.Default(
		server.WithHostPorts(":8082"),
	)

	// 注册路由
	registerRoutes(h, auctionHandler, bidHandler)

	// 启动服务
	log.Println("Auction service starting on :8082")
	h.Spin()
}

// registerRoutes 注册路由
func registerRoutes(h *server.Hertz, auctionHandler *handler.AuctionHandler, bidHandler *handler.BidHandler) {
	v1 := h.Group("/api/v1")

	// 竞拍管理相关路由
	v1.PUT("/auctions/:id/cancel", auctionHandler.Cancel)
	v1.GET("/auctions/:id/result", auctionHandler.GetResult)
	v1.GET("/auctions/:id", auctionHandler.Get)

	// 出价相关路由
	v1.POST("/auctions/:id/bids", bidHandler.PlaceBid)
	v1.GET("/auctions/:id/ranking", bidHandler.GetRanking)
}
