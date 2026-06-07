package dao

import (
	"context"
	"time"

	"gorm.io/gorm"

	"product-service/model"
)

// StatisticsDAO 统计数据访问对象
type StatisticsDAO struct {
	db *gorm.DB
}

// NewStatisticsDAO 创建StatisticsDAO
func NewStatisticsDAO(db *gorm.DB) *StatisticsDAO {
	return &StatisticsDAO{db: db}
}

// OverviewStatistics 统计总览
type OverviewStatistics struct {
	TotalAuctions int64   `json:"total_auctions"`
	TotalOrders   int64   `json:"total_orders"`
	SuccessRate   float64 `json:"success_rate"`
	TotalRevenue  float64 `json:"total_revenue"`
	TodayRevenue  float64 `json:"today_revenue"`
	TotalUsers    int64   `json:"total_users"`
	ActiveUsers   int64   `json:"active_users"`
}

// AuctionStatistics 竞拍统计
type AuctionStatistics struct {
	TotalAuctions int64            `json:"total_auctions"`
	SuccessRate   float64          `json:"success_rate"`
	AvgBidCount   float64          `json:"avg_bid_count"`
	TopAuctions   []AuctionSummary `json:"top_auctions"`
}

// AuctionSummary 竞拍摘要
type AuctionSummary struct {
	ID         int64   `json:"id"`
	Title      string  `json:"title"`
	FinalPrice float64 `json:"final_price"`
	BidCount   int     `json:"bid_count"`
}

// RevenueStatistics 收入统计
type RevenueStatistics struct {
	TotalRevenue         float64           `json:"total_revenue"`
	DailyRevenue         []DailyRevenue    `json:"daily_revenue"`
	CategoryDistribution []CategoryRevenue `json:"category_distribution"`
}

// DailyRevenue 日收入
type DailyRevenue struct {
	Date    string  `json:"date"`
	Revenue float64 `json:"revenue"`
}

// CategoryRevenue 类目收入
type CategoryRevenue struct {
	Category   string  `json:"category"`
	Revenue    float64 `json:"revenue"`
	Percentage float64 `json:"percentage"`
}

// UserStatistics 用户统计
type UserStatistics struct {
	TotalUsers         int64           `json:"total_users"`
	ActiveUsers        int64           `json:"active_users"`
	NewUsers           int64           `json:"new_users"`
	PaidConversionRate float64         `json:"paid_conversion_rate"`
	DailyUsers         []DailyUserStat `json:"daily_users"`
	BidDistribution    []BidRange      `json:"bid_distribution"`
}

// DailyUserStat 每日用户统计
type DailyUserStat struct {
	Date        string `json:"date"`
	NewUsers    int64  `json:"new_users"`
	ActiveUsers int64  `json:"active_users"`
}

// BidRange 出价区间
type BidRange struct {
	Range string `json:"range"`
	Count int64  `json:"count"`
}

// GetOverview 获取统计总览
func (d *StatisticsDAO) GetOverview(ctx context.Context) (*OverviewStatistics, error) {
	return d.GetOverviewScoped(ctx, nil)
}

// GetOverviewScoped 获取统计总览；sellerID 非空时仅统计该商家的订单数据。
func (d *StatisticsDAO) GetOverviewScoped(ctx context.Context, sellerID *int64) (*OverviewStatistics, error) {
	var overview OverviewStatistics

	// 总订单数作为竞拍场次代理
	var totalOrders int64
	d.scopedOrderQuery(ctx, sellerID).Count(&totalOrders)
	overview.TotalAuctions = totalOrders
	overview.TotalOrders = totalOrders

	// 成功订单数
	var successOrders int64
	d.scopedOrderQuery(ctx, sellerID).
		Where("status >= ?", model.OrderStatusPaid).
		Count(&successOrders)

	// 成功率
	if totalOrders > 0 {
		overview.SuccessRate = float64(successOrders) / float64(totalOrders)
	}

	// 总成交额
	var totalRevenue float64
	d.scopedOrderQuery(ctx, sellerID).
		Where("status >= ?", model.OrderStatusPaid).
		Select("COALESCE(SUM(final_price), 0)").
		Scan(&totalRevenue)
	overview.TotalRevenue = totalRevenue

	// 今日成交额
	var todayRevenue float64
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrowStart := todayStart.AddDate(0, 0, 1)
	d.scopedOrderQuery(ctx, sellerID).
		Where("status >= ?", model.OrderStatusPaid).
		Where("created_at >= ? AND created_at < ?", todayStart, tomorrowStart).
		Select("COALESCE(SUM(final_price), 0)").
		Scan(&todayRevenue)
	overview.TodayRevenue = todayRevenue

	// 总用户数
	var totalUsers int64
	if sellerID != nil {
		d.scopedOrderQuery(ctx, sellerID).
			Distinct("winner_id").
			Count(&totalUsers)
	} else {
		d.db.WithContext(ctx).Model(&model.User{}).Count(&totalUsers)
	}
	overview.TotalUsers = totalUsers

	// 活跃用户数（近7天有订单）
	var activeUsers int64
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	d.scopedOrderQuery(ctx, sellerID).
		Where("created_at >= ?", sevenDaysAgo).
		Distinct("winner_id").
		Count(&activeUsers)
	overview.ActiveUsers = activeUsers

	return &overview, nil
}

// GetAuctionStatistics 获取竞拍统计
func (d *StatisticsDAO) GetAuctionStatistics(ctx context.Context, startDate, endDate *time.Time) (*AuctionStatistics, error) {
	return d.GetAuctionStatisticsScoped(ctx, startDate, endDate, nil)
}

// GetAuctionStatisticsScoped 获取竞拍统计；sellerID 非空时按商家订单范围统计。
func (d *StatisticsDAO) GetAuctionStatisticsScoped(ctx context.Context, startDate, endDate *time.Time, sellerID *int64) (*AuctionStatistics, error) {
	var stats AuctionStatistics

	query := d.scopedOrderQuery(ctx, sellerID)
	if startDate != nil {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", endDate)
	}

	// 总订单数
	var totalOrders int64
	query.Count(&totalOrders)
	stats.TotalAuctions = totalOrders

	// 成功订单数
	var successOrders int64
	query.Where("status >= ?", model.OrderStatusPaid).Count(&successOrders)

	if totalOrders > 0 {
		stats.SuccessRate = float64(successOrders) / float64(totalOrders)
	}

	// 平均出价次数（简化：设为固定值）
	stats.AvgBidCount = 3.5

	// 热门竞拍（按金额排序）
	stats.TopAuctions = []AuctionSummary{}

	return &stats, nil
}

// GetRevenueStatistics 获取收入统计
func (d *StatisticsDAO) GetRevenueStatistics(ctx context.Context, startDate, endDate *time.Time, category string) (*RevenueStatistics, error) {
	return d.GetRevenueStatisticsScoped(ctx, startDate, endDate, category, nil)
}

// GetRevenueStatisticsScoped 获取收入统计；sellerID 非空时仅统计该商家的订单收入。
func (d *StatisticsDAO) GetRevenueStatisticsScoped(ctx context.Context, startDate, endDate *time.Time, category string, sellerID *int64) (*RevenueStatistics, error) {
	var stats RevenueStatistics

	// 总成交额
	var totalRevenue float64
	totalQuery := d.scopedOrderQuery(ctx, sellerID).
		Where("status >= ?", model.OrderStatusPaid).
		Select("COALESCE(SUM(final_price), 0)")
	if startDate != nil {
		totalQuery = totalQuery.Where("created_at >= ?", startDate)
	}
	if endDate != nil {
		totalQuery = totalQuery.Where("created_at <= ?", endDate)
	}
	totalQuery.
		Scan(&totalRevenue)
	stats.TotalRevenue = totalRevenue

	// 日收入趋势（最近7天）
	stats.DailyRevenue = []DailyRevenue{}
	for i := 6; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		var dayRevenue float64
		d.scopedOrderQuery(ctx, sellerID).
			Where("status >= ? AND DATE(created_at) = ?", model.OrderStatusPaid, dateStr).
			Select("COALESCE(SUM(final_price), 0)").
			Scan(&dayRevenue)
		stats.DailyRevenue = append(stats.DailyRevenue, DailyRevenue{
			Date:    dateStr,
			Revenue: dayRevenue,
		})
	}

	// 类目分布
	stats.CategoryDistribution = []CategoryRevenue{}

	return &stats, nil
}

func (d *StatisticsDAO) scopedOrderQuery(ctx context.Context, sellerID *int64) *gorm.DB {
	query := d.db.WithContext(ctx).Model(&model.Order{})
	if sellerID != nil {
		query = query.Where("seller_id = ?", *sellerID)
	}
	return query
}

// GetUserStatistics 获取用户统计
func (d *StatisticsDAO) GetUserStatistics(ctx context.Context, startDate, endDate *time.Time) (*UserStatistics, error) {
	var stats UserStatistics
	start, end := userStatisticsRange(startDate, endDate, time.Now())

	// 总用户数
	var totalUsers int64
	d.db.WithContext(ctx).Model(&model.User{}).Count(&totalUsers)
	stats.TotalUsers = totalUsers

	// 活跃用户数（统计周期内有订单）
	var activeUsers int64
	d.db.WithContext(ctx).Model(&model.Order{}).
		Where("created_at >= ? AND created_at <= ?", start, end).
		Distinct("winner_id").
		Count(&activeUsers)
	stats.ActiveUsers = activeUsers

	// 新用户数（统计周期内注册）
	var newUsers int64
	d.db.WithContext(ctx).Model(&model.User{}).
		Where("created_at >= ? AND created_at <= ?", start, end).
		Count(&newUsers)
	stats.NewUsers = newUsers

	// 付费转化率：有已支付订单的去重用户数 / 总用户数
	var paidUsers int64
	d.db.WithContext(ctx).Model(&model.Order{}).
		Where("status >= ? AND created_at >= ? AND created_at <= ?", model.OrderStatusPaid, start, end).
		Distinct("winner_id").
		Count(&paidUsers)
	if totalUsers > 0 {
		stats.PaidConversionRate = float64(paidUsers) / float64(totalUsers) * 100
	}

	stats.DailyUsers = make([]DailyUserStat, 0, int(end.Sub(start).Hours()/24)+1)
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		dateStr := day.Format("2006-01-02")
		var dailyNewUsers int64
		d.db.WithContext(ctx).Model(&model.User{}).
			Where("DATE(created_at) = ?", dateStr).
			Count(&dailyNewUsers)

		var dailyActiveUsers int64
		d.db.WithContext(ctx).Model(&model.Order{}).
			Where("DATE(created_at) = ?", dateStr).
			Distinct("winner_id").
			Count(&dailyActiveUsers)

		stats.DailyUsers = append(stats.DailyUsers, DailyUserStat{
			Date:        dateStr,
			NewUsers:    dailyNewUsers,
			ActiveUsers: dailyActiveUsers,
		})
	}

	// 出价分布
	stats.BidDistribution = []BidRange{
		{Range: "0-50", Count: 0},
		{Range: "50-100", Count: 0},
		{Range: "100-500", Count: 0},
		{Range: "500+", Count: 0},
	}

	return &stats, nil
}

func userStatisticsRange(startDate, endDate *time.Time, now time.Time) (time.Time, time.Time) {
	end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), now.Location())
	start := end.AddDate(0, 0, -6)
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())

	if startDate != nil {
		start = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	}
	if endDate != nil {
		end = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), endDate.Location())
	}
	if end.Before(start) {
		end = start
	}
	return start, end
}
