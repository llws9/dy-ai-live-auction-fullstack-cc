package service

import (
	"context"
	"time"

	"product-service/dao"
)

// StatisticsService 统计服务
type StatisticsService struct {
	statisticsDAO *dao.StatisticsDAO
}

// NewStatisticsService 创建统计服务
func NewStatisticsService(statisticsDAO *dao.StatisticsDAO) *StatisticsService {
	return &StatisticsService{
		statisticsDAO: statisticsDAO,
	}
}

// GetOverview 获取统计总览
// @Summary 获取统计总览
// @Description 获取管理后台首页大屏的统计数据
// @Tags statistics
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dao.OverviewStatistics
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /statistics/overview [get]
func (s *StatisticsService) GetOverview(ctx context.Context) (*dao.OverviewStatistics, error) {
	return s.GetOverviewScoped(ctx, nil)
}

// GetOverviewScoped 获取统计总览；sellerID 非空时仅统计该商家的订单数据。
func (s *StatisticsService) GetOverviewScoped(ctx context.Context, sellerID *int64) (*dao.OverviewStatistics, error) {
	return s.statisticsDAO.GetOverviewScoped(ctx, sellerID)
}

// GetAuctionStatistics 获取竞拍统计
// @Summary 获取竞拍统计
// @Description 获取竞拍相关的统计数据
// @Tags statistics
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "开始日期"
// @Param end_date query string false "结束日期"
// @Success 200 {object} dao.AuctionStatistics
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /statistics/auctions [get]
func (s *StatisticsService) GetAuctionStatistics(ctx context.Context, startDate, endDate *time.Time) (*dao.AuctionStatistics, error) {
	return s.GetAuctionStatisticsScoped(ctx, startDate, endDate, nil)
}

// GetAuctionStatisticsScoped 获取竞拍统计；sellerID 非空时按商家订单范围统计。
func (s *StatisticsService) GetAuctionStatisticsScoped(ctx context.Context, startDate, endDate *time.Time, sellerID *int64) (*dao.AuctionStatistics, error) {
	return s.statisticsDAO.GetAuctionStatisticsScoped(ctx, startDate, endDate, sellerID)
}

// GetRevenueStatistics 获取收入统计
// @Summary 获取收入统计
// @Description 获取收入相关的统计数据
// @Tags statistics
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "开始日期"
// @Param end_date query string false "结束日期"
// @Param category query string false "商品类目"
// @Success 200 {object} dao.RevenueStatistics
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /statistics/revenue [get]
func (s *StatisticsService) GetRevenueStatistics(ctx context.Context, startDate, endDate *time.Time, category string) (*dao.RevenueStatistics, error) {
	return s.GetRevenueStatisticsScoped(ctx, startDate, endDate, category, nil)
}

// GetRevenueStatisticsScoped 获取收入统计；sellerID 非空时仅统计该商家的订单收入。
func (s *StatisticsService) GetRevenueStatisticsScoped(ctx context.Context, startDate, endDate *time.Time, category string, sellerID *int64) (*dao.RevenueStatistics, error) {
	return s.statisticsDAO.GetRevenueStatisticsScoped(ctx, startDate, endDate, category, sellerID)
}

// GetUserStatistics 获取用户统计
// @Summary 获取用户统计
// @Description 获取用户相关的统计数据
// @Tags statistics
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "开始日期"
// @Param end_date query string false "结束日期"
// @Success 200 {object} dao.UserStatistics
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /statistics/users [get]
func (s *StatisticsService) GetUserStatistics(ctx context.Context, startDate, endDate *time.Time) (*dao.UserStatistics, error) {
	return s.statisticsDAO.GetUserStatistics(ctx, startDate, endDate)
}
