package main

import (
	"fmt"
	"math/rand"
	"time"

	"product-service/model"

	"github.com/shopspring/decimal"
)

var (
	// 类别预设数据
	categoryNames = []string{
		"数码电子", "服装配饰", "家居生活", "美妆护肤",
		"食品饮料", "运动户外", "母婴用品", "珠宝首饰",
		"图书文具", "汽车用品", "宠物用品", "艺术品",
	}
	categoryCodes = []string{
		"ELECTRONICS", "CLOTHING", "HOME", "BEAUTY",
		"FOOD", "SPORTS", "BABY", "JEWELRY",
		"BOOKS", "AUTOS", "PET", "ART",
	}
	categoryDescriptions = []string{
		"智能手机、电脑、数码配件等电子产品",
		"男装、女装、鞋帽、箱包等服饰配件",
		"家具、家电、厨具、装饰品等生活用品",
		"化妆品、护肤品、香水等美容产品",
		"零食、饮料、生鲜、保健品等食品",
		"运动器材、户外装备、健身用品",
		"婴儿用品、童装、玩具、孕产用品",
		"黄金、钻石、翡翠、珍珠等珠宝首饰",
		"书籍、杂志、文具、办公用品",
		"汽车配件、车载用品、保养工具",
		"宠物食品、宠物用品、宠物玩具",
		"字画、雕塑、收藏品、工艺品",
	}

	// 用户名前缀
	userNamePrefixes = []string{
		"张", "王", "李", "刘", "陈", "杨", "黄", "赵", "周", "吴",
		"徐", "孙", "马", "朱", "胡", "郭", "何", "林", "高", "罗",
	}
	userNameSuffixes = []string{
		"明", "华", "强", "伟", "磊", "洋", "勇", "军", "杰", "涛",
		"婷", "静", "丽", "芳", "燕", "娟", "英", "梅", "琳", "颖",
	}

	// 直播间名称模板
	liveStreamNames = []string{
		"今日好物推荐", "新品首发专场", "限时秒杀直播",
		"品牌特惠专场", "爆款清仓直播", "品质生活分享",
		"潮流穿搭指南", "美食推荐专场", "数码好物测评",
		"居家好物分享", "美妆技巧分享", "珠宝鉴赏专场",
	}
	liveStreamDescriptions = []string{
		"每日精选优质商品，为您带来实惠好物",
		"新品首发，抢先体验最新产品",
		"限时秒杀，错过就要等下次了",
		"品牌官方特惠，品质保障",
		"爆款清仓，超值优惠不容错过",
		"品质生活分享，提升生活品质",
		"潮流穿搭指南，教你如何搭配",
		"美食推荐专场，精选各地美食",
		"数码好物测评，专业评测推荐",
		"居家好物分享，打造温馨家园",
		"美妆技巧分享，教你化妆技巧",
		"珠宝鉴赏专场，专业珠宝讲解",
	}
)

// GenerateCategories 生成类别数据 (T025)
func GenerateCategories(cfg *SeedConfig) []model.Category {
	categories := make([]model.Category, 0, cfg.CategoriesCount)

	for i := 0; i < cfg.CategoriesCount; i++ {
		category := model.Category{
			Name:        categoryNames[i],
			Code:        categoryCodes[i],
			Description: categoryDescriptions[i],
			SortOrder:   i,
			Status:      model.CategoryStatusActive,
		}
		categories = append(categories, category)
	}

	return categories
}

// GenerateUsers 生成用户数据 (T026)
func GenerateUsers(cfg *SeedConfig) []model.User {
	users := make([]model.User, 0, cfg.UsersCount)

	now := time.Now()
	r := rand.New(rand.NewSource(now.UnixNano()))

	// 计算各角色数量（角色：0=普通用户, 1=主播, 2=管理员）
	adminCount := int(float64(cfg.UsersCount) * cfg.AdminRatio)
	streamerCount := int(float64(cfg.UsersCount) * cfg.StreamerRatio)
	normalCount := cfg.UsersCount - streamerCount - adminCount

	// 生成管理员
	for i := 0; i < adminCount; i++ {
		email := fmt.Sprintf("admin_%d@example.com", i+1)
		phone := fmt.Sprintf("138%08d", r.Intn(100000000))
		user := model.User{
			Name:     fmt.Sprintf("%s%s管理员", userNamePrefixes[r.Intn(len(userNamePrefixes))], userNameSuffixes[r.Intn(len(userNameSuffixes))]),
			Avatar:   fmt.Sprintf("https://example.com/avatars/admin_%d.jpg", i+1),
			Email:    &email,
			Phone:    &phone,
			Password: "password123_hash",   // 简化，实际应用需要hash
			Role:     int(model.RoleAdmin), // 2
			Status:   1,                    // 正常
		}
		users = append(users, user)
	}

	// 生成主播
	for i := 0; i < streamerCount; i++ {
		email := fmt.Sprintf("streamer_%d@example.com", i+1)
		phone := fmt.Sprintf("158%08d", r.Intn(100000000))
		user := model.User{
			Name:     fmt.Sprintf("%s%s主播", userNamePrefixes[r.Intn(len(userNamePrefixes))], userNameSuffixes[r.Intn(len(userNameSuffixes))]),
			Avatar:   fmt.Sprintf("https://example.com/avatars/streamer_%d.jpg", i+1),
			Email:    &email,
			Phone:    &phone,
			Password: "password123_hash",
			Role:     int(model.RoleStreamer), // 1
			Status:   1,
		}
		users = append(users, user)
	}

	// 生成普通用户
	for i := 0; i < normalCount; i++ {
		email := fmt.Sprintf("user_%d@example.com", i+1)
		phone := fmt.Sprintf("186%08d", r.Intn(100000000))
		user := model.User{
			Name:     fmt.Sprintf("%s%s", userNamePrefixes[r.Intn(len(userNamePrefixes))], userNameSuffixes[r.Intn(len(userNameSuffixes))]),
			Avatar:   fmt.Sprintf("https://example.com/avatars/user_%d.jpg", i+1),
			Email:    &email,
			Phone:    &phone,
			Password: "password123_hash",
			Role:     int(model.RoleUser), // 0
			Status:   1,
		}
		users = append(users, user)
	}

	return users
}

// GenerateProducts 生成商品数据 (T027)
func GenerateProducts(cfg *SeedConfig, users []model.User, categories []model.Category) []model.Product {
	products := make([]model.Product, 0, cfg.ProductsCount)

	now := time.Now()
	r := rand.New(rand.NewSource(now.UnixNano() + 1))

	// 筛选主播和管理员用户（可以管理商品）
	managers := make([]model.User, 0)
	for _, u := range users {
		if u.Role >= int(model.RoleStreamer) {
			managers = append(managers, u)
		}
	}

	// 计算各状态数量
	draftCount := int(float64(cfg.ProductsCount) * cfg.DraftRatio)
	publishedCount := int(float64(cfg.ProductsCount) * cfg.PublishedRatio)
	unpublishedCount := cfg.ProductsCount - draftCount - publishedCount

	// 生成草稿商品
	for i := 0; i < draftCount; i++ {
		category := categories[r.Intn(len(categories))]
		images := []string{
			fmt.Sprintf("https://example.com/products/product_%d_1.jpg", i+1),
			fmt.Sprintf("https://example.com/products/product_%d_2.jpg", i+1),
		}

		product := model.Product{
			Name:        fmt.Sprintf("商品_%d", i+1),
			Description: fmt.Sprintf("这是一款%s类商品，编号%d，品质优良", category.Name, i+1),
			Images:      model.JSONArray(images),
			CategoryID:  &category.ID,
			Status:      model.ProductStatusDraft,
		}
		products = append(products, product)
	}

	// 生成已发布商品
	for i := 0; i < publishedCount; i++ {
		category := categories[r.Intn(len(categories))]
		images := []string{
			fmt.Sprintf("https://example.com/products/product_%d_1.jpg", draftCount+i+1),
			fmt.Sprintf("https://example.com/products/product_%d_2.jpg", draftCount+i+1),
			fmt.Sprintf("https://example.com/products/product_%d_3.jpg", draftCount+i+1),
		}

		product := model.Product{
			Name:        fmt.Sprintf("商品_%d", draftCount+i+1),
			Description: fmt.Sprintf("这是一款%s类商品，编号%d，品质优良，值得购买", category.Name, draftCount+i+1),
			Images:      model.JSONArray(images),
			CategoryID:  &category.ID,
			Status:      model.ProductStatusPublished,
		}
		products = append(products, product)
	}

	// 生成已下架商品
	for i := 0; i < unpublishedCount; i++ {
		category := categories[r.Intn(len(categories))]
		images := []string{
			fmt.Sprintf("https://example.com/products/product_%d_1.jpg", draftCount+publishedCount+i+1),
		}

		product := model.Product{
			Name:        fmt.Sprintf("商品_%d", draftCount+publishedCount+i+1),
			Description: fmt.Sprintf("这是一款%s类商品，编号%d，已下架", category.Name, draftCount+publishedCount+i+1),
			Images:      model.JSONArray(images),
			CategoryID:  &category.ID,
			Status:      model.ProductStatusUnpublished,
		}
		products = append(products, product)
	}

	return products
}

// GenerateLiveStreams 生成直播间数据 (T028)
func GenerateLiveStreams(cfg *SeedConfig, users []model.User) []model.LiveStream {
	liveStreams := make([]model.LiveStream, 0, cfg.LiveStreamsCount)

	now := time.Now()
	r := rand.New(rand.NewSource(now.UnixNano() + 2))

	// 筛选主播用户（CreatorID = 主播ID）
	streamers := make([]model.User, 0)
	for _, u := range users {
		if u.Role == int(model.RoleStreamer) {
			streamers = append(streamers, u)
		}
	}

	for i := 0; i < cfg.LiveStreamsCount; i++ {
		streamer := streamers[r.Intn(len(streamers))]
		// 状态随机：0=禁用, 1=正常
		status := model.LiveStreamStatus(r.Intn(2))

		liveStream := model.LiveStream{
			CreatorID:   streamer.ID,
			Name:        fmt.Sprintf("%s_%d", liveStreamNames[i%len(liveStreamNames)], i+1),
			Description: liveStreamDescriptions[i%len(liveStreamDescriptions)],
			CoverImage:  fmt.Sprintf("https://example.com/live_streams/live_%d.jpg", i+1),
			Status:      status,
		}
		liveStreams = append(liveStreams, liveStream)
	}

	return liveStreams
}

// GenerateAuctionRules 生成竞拍规则数据 (T029)
func GenerateAuctionRules(cfg *SeedConfig, products []model.Product) []model.AuctionRule {
	auctionRules := make([]model.AuctionRule, 0, cfg.AuctionRulesCount)

	now := time.Now()
	r := rand.New(rand.NewSource(now.UnixNano() + 3))

	// 筛选已发布的商品
	publishedProducts := make([]model.Product, 0)
	for _, p := range products {
		if p.Status == model.ProductStatusPublished {
			publishedProducts = append(publishedProducts, p)
		}
	}

	for i := 0; i < cfg.AuctionRulesCount; i++ {
		product := publishedProducts[r.Intn(len(publishedProducts))]

		// 起拍价：随机设定
		startPrice := float64(50 + r.Intn(500))
		// 加价幅度：5-50元
		increment := float64(5 + r.Intn(45))
		// 封顶价（可选）
		var capPrice *float64
		if r.Intn(2) == 0 {
			cp := startPrice * 3
			capPrice = &cp
		}
		// 竞拍时长：5-15分钟
		duration := 300 + r.Intn(600)

		auctionRule := model.AuctionRule{
			ProductID:          product.ID,
			StartPrice:         startPrice,
			Increment:          increment,
			CapPrice:           capPrice,
			Duration:           duration,
			DelayDuration:      30,
			MaxDelayTime:       180,
			TriggerDelayBefore: 30,
		}
		auctionRules = append(auctionRules, auctionRule)
	}

	return auctionRules
}

// GenerateOrders 生成订单数据 (T032)
func GenerateOrders(cfg *SeedConfig, users []model.User, products []model.Product) []model.Order {
	orders := make([]model.Order, 0, cfg.OrdersCount)

	now := time.Now()
	r := rand.New(rand.NewSource(now.UnixNano() + 6))

	// 筛选普通用户（买家/获胜者）
	buyers := make([]model.User, 0)
	for _, u := range users {
		if u.Role == int(model.RoleUser) {
			buyers = append(buyers, u)
		}
	}

	// 计算各状态数量
	pendingCount := int(float64(cfg.OrdersCount) * cfg.UnpaidRatio)
	paidCount := int(float64(cfg.OrdersCount) * cfg.PaidRatio)
	shippedCount := int(float64(cfg.OrdersCount) * cfg.ShippedRatio)
	completedCount := cfg.OrdersCount - pendingCount - paidCount - shippedCount

	// 生成待支付订单
	for i := 0; i < pendingCount; i++ {
		buyer := buyers[r.Intn(len(buyers))]
		product := products[r.Intn(len(products))]
		finalPrice := decimal.NewFromInt(int64(50 + r.Intn(500)))

		order := model.Order{
			AuctionID:  int64(i + 1), // 假设竞拍ID从1开始
			ProductID:  product.ID,
			WinnerID:   buyer.ID,
			FinalPrice: finalPrice,
			Status:     model.OrderStatusPending,
		}
		orders = append(orders, order)
	}

	// 生成已支付订单
	for i := 0; i < paidCount; i++ {
		buyer := buyers[r.Intn(len(buyers))]
		product := products[r.Intn(len(products))]
		finalPrice := decimal.NewFromInt(int64(50 + r.Intn(500)))
		paidAt := now.Add(-time.Hour * time.Duration(r.Intn(24)))

		order := model.Order{
			AuctionID:  int64(pendingCount + i + 1),
			ProductID:  product.ID,
			WinnerID:   buyer.ID,
			FinalPrice: finalPrice,
			Status:     model.OrderStatusPaid,
			PaidAt:     &paidAt,
		}
		orders = append(orders, order)
	}

	// 生成已发货订单
	for i := 0; i < shippedCount; i++ {
		buyer := buyers[r.Intn(len(buyers))]
		product := products[r.Intn(len(products))]
		finalPrice := decimal.NewFromInt(int64(50 + r.Intn(500)))
		paidAt := now.Add(-time.Hour * 48)
		shippedAt := now.Add(-time.Hour * time.Duration(r.Intn(24)))

		order := model.Order{
			AuctionID:  int64(pendingCount + paidCount + i + 1),
			ProductID:  product.ID,
			WinnerID:   buyer.ID,
			FinalPrice: finalPrice,
			Status:     model.OrderStatusShipped,
			PaidAt:     &paidAt,
			ShippedAt:  &shippedAt,
		}
		orders = append(orders, order)
	}

	// 生成已完成订单
	for i := 0; i < completedCount; i++ {
		buyer := buyers[r.Intn(len(buyers))]
		product := products[r.Intn(len(products))]
		finalPrice := decimal.NewFromInt(int64(50 + r.Intn(500)))
		paidAt := now.Add(-time.Hour * 72)
		shippedAt := now.Add(-time.Hour * 48)
		completedAt := now.Add(-time.Hour * time.Duration(r.Intn(12)))

		order := model.Order{
			AuctionID:   int64(pendingCount + paidCount + shippedCount + i + 1),
			ProductID:   product.ID,
			WinnerID:    buyer.ID,
			FinalPrice:  finalPrice,
			Status:      model.OrderStatusCompleted,
			PaidAt:      &paidAt,
			ShippedAt:   &shippedAt,
			CompletedAt: &completedAt,
		}
		orders = append(orders, order)
	}

	return orders
}

// GenerateAuctions 和 GenerateBids、GenerateNotifications 需要 auction-service 的模型
// 这些数据需要在 auction-service 的数据库中生成，这里只提供结构说明
// 实际生产中应单独为 auction-service 编写 seed 脚本

// Note: The auction, bid, and notification generation functions are kept as
// documentation for what would need to be generated in the auction-service database.
// Since those models are defined in a different service, this seed script focuses
// on product-service data: categories, users, products, live_streams, auction_rules, orders.
