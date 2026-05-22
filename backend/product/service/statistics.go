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
	return s.statisticsDAO.GetOverview(ctx)
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
	return s.statisticsDAO.GetAuctionStatistics(ctx, startDate, endDate)
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
	return s.statisticsDAO.GetRevenueStatistics(ctx, startDate, endDate, category)
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
