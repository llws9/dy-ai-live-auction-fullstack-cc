package main

// SeedConfig 测试数据生成配置
type SeedConfig struct {
	// 数据数量配置
	CategoriesCount   int // 类别数量
	UsersCount        int // 用户数量（包含各角色）
	ProductsCount     int // 商品数量
	LiveStreamsCount  int // 直播间数量
	AuctionRulesCount int // 竞拍规则数量
	OrdersCount       int // 订单数量

	// 数据分布配置
	StreamerRatio float64 // 主播比例（占总用户数）
	AdminRatio    float64 // 管理员比例（占总用户数）
	// 普通用户比例 = 1 - StreamerRatio - AdminRatio

	// 商品状态分布（ProductStatus: Draft, Published, Unpublished）
	DraftRatio      float64 // 草稿状态比例
	PublishedRatio  float64 // 已发布状态比例
	UnpublishedRatio float64 // 已下架状态比例

	// 订单状态分布（OrderStatus: Pending, Paid, Shipped, Completed）
	UnpaidRatio    float64 // 待支付状态比例
	PaidRatio      float64 // 已支付状态比例
	ShippedRatio   float64 // 已发货状态比例
	CompletedRatio float64 // 已完成状态比例
}

// DefaultConfig 返回默认配置（中等规模）
func DefaultConfig() *SeedConfig {
	return &SeedConfig{
		// 数据数量
		CategoriesCount:   8,  // 8个类别
		UsersCount:        50, // 50个用户
		ProductsCount:     30, // 30个商品
		LiveStreamsCount:  10, // 10个直播间
		AuctionRulesCount: 20, // 20个竞拍规则
		OrdersCount:       30, // 30个订单

		// 用户角色分布
		StreamerRatio: 0.2, // 20%主播
		AdminRatio:    0.1, // 10%管理员
		// 普通用户：70%

		// 商品状态分布
		DraftRatio:      0.3, // 30%草稿
		PublishedRatio:  0.6, // 60%已发布
		UnpublishedRatio: 0.1, // 10%已下架

		// 订单状态分布
		UnpaidRatio:    0.2, // 20%待支付
		PaidRatio:      0.3, // 30%已支付
		ShippedRatio:   0.3, // 30%已发货
		CompletedRatio: 0.2, // 20%已完成
	}
}

// SmallConfig 小规模配置（约100条数据）
func SmallConfig() *SeedConfig {
	return &SeedConfig{
		CategoriesCount:   5,
		UsersCount:        20,
		ProductsCount:     15,
		LiveStreamsCount:  5,
		AuctionRulesCount: 10,
		OrdersCount:       10,

		StreamerRatio: 0.25,
		AdminRatio:    0.15,

		DraftRatio:      0.4,
		PublishedRatio:  0.5,
		UnpublishedRatio: 0.1,

		UnpaidRatio:    0.25,
		PaidRatio:      0.25,
		ShippedRatio:   0.25,
		CompletedRatio: 0.25,
	}
}

// LargeConfig 大规模配置（约500条数据）
func LargeConfig() *SeedConfig {
	return &SeedConfig{
		CategoriesCount:   12,
		UsersCount:        100,
		ProductsCount:     100,
		LiveStreamsCount:  20,
		AuctionRulesCount: 50,
		OrdersCount:       100,

		StreamerRatio: 0.15,
		AdminRatio:    0.05,

		DraftRatio:      0.2,
		PublishedRatio:  0.7,
		UnpublishedRatio: 0.1,

		UnpaidRatio:    0.15,
		PaidRatio:      0.25,
		ShippedRatio:   0.35,
		CompletedRatio: 0.25,
	}
}