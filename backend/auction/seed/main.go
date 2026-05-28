package main

import (
	"flag"
	"fmt"
	"log"

	"auction-service/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	size := flag.String("size", "medium", "数据规模: small, medium, large")
	dbHost := flag.String("db-host", "localhost", "数据库地址")
	dbPort := flag.Int("db-port", 3306, "数据库端口")
	dbUser := flag.String("db-user", "root", "数据库用户")
	dbPassword := flag.String("db-password", "", "数据库密码")
	dbName := flag.String("db-name", "live_auction", "数据库名称")

	flag.Parse()

	cfg := GetDefaultConfig(*size)
	log.Printf("开始生成种子数据，规模: %s", *size)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		*dbUser, *dbPassword, *dbHost, *dbPort, *dbName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	var productIDs, liveStreamIDs, userIDs, auctionIDs []int64

	db.Model(&model.User{}).Pluck("id", &userIDs)
	if len(userIDs) == 0 {
		log.Fatal("请先运行product-service的seed脚本生成用户数据")
	}
	log.Printf("获取到 %d 个用户", len(userIDs))

	db.Table("live_streams").Pluck("id", &liveStreamIDs)
	log.Printf("获取到 %d 个直播间", len(liveStreamIDs))

	db.Table("products").Where("status = ?", 1).Pluck("id", &productIDs)
	log.Printf("获取到 %d 个已发布商品", len(productIDs))

	log.Println("生成竞拍数据...")
	auctions := GenerateAuctions(cfg, productIDs, liveStreamIDs, userIDs)
	if err := db.CreateInBatches(&auctions, 100).Error; err != nil {
		log.Fatalf("创建竞拍数据失败: %v", err)
	}
	log.Printf("成功创建 %d 条竞拍数据", len(auctions))

	db.Model(&model.Auction{}).Pluck("id", &auctionIDs)

	log.Println("生成出价数据...")
	bids := GenerateBids(cfg, auctions, userIDs)
	if err := db.CreateInBatches(&bids, 100).Error; err != nil {
		log.Fatalf("创建出价数据失败: %v", err)
	}
	log.Printf("成功创建 %d 条出价数据", len(bids))

	log.Println("生成通知数据...")
	notifications := GenerateNotifications(cfg, userIDs, auctionIDs)
	if err := db.CreateInBatches(&notifications, 100).Error; err != nil {
		log.Fatalf("创建通知数据失败: %v", err)
	}
	log.Printf("成功创建 %d 条通知数据", len(notifications))

	log.Println("生成点天灯订阅数据...")
	skyLamps := GenerateSkyLampSubscriptions(cfg, auctionIDs, userIDs)
	if err := db.CreateInBatches(&skyLamps, 100).Error; err != nil {
		log.Fatalf("创建点天灯数据失败: %v", err)
	}
	log.Printf("成功创建 %d 条点天灯订阅", len(skyLamps))

	log.Println("生成用户关注直播间数据...")
	follows := GenerateUserLiveStreamFollows(cfg, userIDs, liveStreamIDs)
	if err := db.CreateInBatches(&follows, 100).Error; err != nil {
		log.Fatalf("创建关注数据失败: %v", err)
	}
	log.Printf("成功创建 %d 条关注数据", len(follows))

	log.Println("生成商品提醒订阅数据...")
	reminders := GenerateUserProductReminders(cfg, userIDs, productIDs, auctionIDs)
	if err := db.CreateInBatches(&reminders, 100).Error; err != nil {
		log.Fatalf("创建提醒数据失败: %v", err)
	}
	log.Printf("成功创建 %d 条商品提醒订阅", len(reminders))

	log.Println("种子数据生成完成！")
	log.Println("数据统计:")
	log.Printf("  - 竞拍: %d", len(auctions))
	log.Printf("  - 出价: %d", len(bids))
	log.Printf("  - 通知: %d", len(notifications))
	log.Printf("  - 点天灯订阅: %d", len(skyLamps))
	log.Printf("  - 关注直播间: %d", len(follows))
	log.Printf("  - 商品提醒: %d", len(reminders))
}
