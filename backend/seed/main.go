package main

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"

	"product-service/dao"
	"product-service/model"
)

func main() {
	// 初始化数据库连接
	db, err := dao.InitDBFromEnv()
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}

	// 使用默认配置（中等规模）
	cfg := DefaultConfig()
	log.Printf("Seed configuration: %+v", cfg)

	ctx := context.Background()

	// 开始生成数据
	log.Println("Starting seed data generation...")
	startTime := time.Now()

	// T025: 生成类别数据
	log.Println("Generating categories...")
	categories := GenerateCategories(cfg)
	if err := insertCategories(ctx, db, categories); err != nil {
		log.Printf("Warning: Failed to insert categories: %v", err)
	} else {
		log.Printf("Generated %d categories", len(categories))
	}

	// T026: 生成用户数据
	log.Println("Generating users...")
	users := GenerateUsers(cfg)
	if err := insertUsers(ctx, db, users); err != nil {
		log.Printf("Warning: Failed to insert users: %v", err)
	} else {
		log.Printf("Generated %d users", len(users))
	}

	// T027: 生成商品数据
	log.Println("Generating products...")
	products := GenerateProducts(cfg, users, categories)
	if err := insertProducts(ctx, db, products); err != nil {
		log.Printf("Warning: Failed to insert products: %v", err)
	} else {
		log.Printf("Generated %d products", len(products))
	}

	// T028: 生成直播间数据
	log.Println("Generating live streams...")
	liveStreams := GenerateLiveStreams(cfg, users)
	if err := insertLiveStreams(ctx, db, liveStreams); err != nil {
		log.Printf("Warning: Failed to insert live streams: %v", err)
	} else {
		log.Printf("Generated %d live streams", len(liveStreams))
	}

	// T029: 生成竞拍规则数据
	log.Println("Generating auction rules...")
	auctionRules := GenerateAuctionRules(cfg, products)
	if err := insertAuctionRules(ctx, db, auctionRules); err != nil {
		log.Printf("Warning: Failed to insert auction rules: %v", err)
	} else {
		log.Printf("Generated %d auction rules", len(auctionRules))
	}

	// T032: 生成订单数据
	log.Println("Generating orders...")
	orders := GenerateOrders(cfg, users, products)
	if err := insertOrders(ctx, db, orders); err != nil {
		log.Printf("Warning: Failed to insert orders: %v", err)
	} else {
		log.Printf("Generated %d orders", len(orders))
	}

	// T030-T031, T033: 竞拍、出价、通知数据需要在 auction-service 数据库中生成
	log.Println("Note: Auctions, bids, and notifications need to be generated in auction-service database separately")

	// 完成
	duration := time.Since(startTime)
	log.Printf("Seed data generation completed in %v", duration)
	log.Printf("Summary:")
	log.Printf("  - Categories: %d", len(categories))
	log.Printf("  - Users: %d (Admin: %d, Streamer: %d, User: rest)",
		countUsers(users, int(model.RoleAdmin)), countUsers(users, int(model.RoleStreamer)))
	log.Printf("  - Products: %d (Draft: %d, Published: %d, Unpublished: %d)",
		countProducts(products, model.ProductStatusDraft), countProducts(products, model.ProductStatusPublished), countProducts(products, model.ProductStatusUnpublished))
	log.Printf("  - Live Streams: %d", len(liveStreams))
	log.Printf("  - Auction Rules: %d", len(auctionRules))
	log.Printf("  - Orders: %d (Pending: %d, Paid: %d, Shipped: %d, Completed: %d)",
		countOrders(orders, model.OrderStatusPending), countOrders(orders, model.OrderStatusPaid),
		countOrders(orders, model.OrderStatusShipped), countOrders(orders, model.OrderStatusCompleted))
}

// insertCategories 批量插入类别
func insertCategories(ctx context.Context, db *gorm.DB, categories []model.Category) error {
	return db.WithContext(ctx).CreateInBatches(categories, 100).Error
}

// insertUsers 批量插入用户
func insertUsers(ctx context.Context, db *gorm.DB, users []model.User) error {
	return db.WithContext(ctx).CreateInBatches(users, 100).Error
}

// insertProducts 批量插入商品
func insertProducts(ctx context.Context, db *gorm.DB, products []model.Product) error {
	return db.WithContext(ctx).CreateInBatches(products, 100).Error
}

// insertLiveStreams 批量插入直播间
func insertLiveStreams(ctx context.Context, db *gorm.DB, liveStreams []model.LiveStream) error {
	return db.WithContext(ctx).CreateInBatches(liveStreams, 100).Error
}

// insertAuctionRules 批量插入竞拍规则
func insertAuctionRules(ctx context.Context, db *gorm.DB, auctionRules []model.AuctionRule) error {
	return db.WithContext(ctx).CreateInBatches(auctionRules, 100).Error
}

// insertOrders 批量插入订单
func insertOrders(ctx context.Context, db *gorm.DB, orders []model.Order) error {
	return db.WithContext(ctx).CreateInBatches(orders, 100).Error
}

// countUsers 统计特定角色的用户数量
func countUsers(users []model.User, role int) int {
	count := 0
	for _, u := range users {
		if u.Role == role {
			count++
		}
	}
	return count
}

// countProducts 统计特定状态的商品数量
func countProducts(products []model.Product, status model.ProductStatus) int {
	count := 0
	for _, p := range products {
		if p.Status == status {
			count++
		}
	}
	return count
}

// countOrders 统计特定状态的订单数量
func countOrders(orders []model.Order, status model.OrderStatus) int {
	count := 0
	for _, o := range orders {
		if o.Status == status {
			count++
		}
	}
	return count
}