package handler

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"product-service/service"
)

// StatisticsHandler 统计 Handler
type StatisticsHandler struct {
	statisticsService *service.StatisticsService
}

// NewStatisticsHandler 创建统计 Handler
func NewStatisticsHandler(statisticsService *service.StatisticsService) *StatisticsHandler {
	return &StatisticsHandler{
		statisticsService: statisticsService,
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
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /statistics/overview [get]
func (h *StatisticsHandler) GetOverview(ctx context.Context, c *app.RequestContext) {
	// 权限检查：仅管理员可访问
	role, exists := c.Get("role")
	if !exists || role.(int) != 2 { // RoleAdmin = 2
		c.JSON(403, map[string]interface{}{
			"code":    403,
			"message": "无权限访问",
		})
		return
	}

	overview, err := h.statisticsService.GetOverview(ctx)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取统计总览失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, overview)
}

// GetAuctionStatistics 获取竞拍统计
// @Summary 获取竞拍统计
// @Description 获取竞拍相关的统计数据
// @Tags statistics
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "开始日期 (YYYY-MM-DD)"
// @Param end_date query string false "结束日期 (YYYY-MM-DD)"
// @Success 200 {object} dao.AuctionStatistics
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /statistics/auctions [get]
func (h *StatisticsHandler) GetAuctionStatistics(ctx context.Context, c *app.RequestContext) {
	// 权限检查
	role, exists := c.Get("role")
	if !exists || role.(int) != 2 {
		c.JSON(403, map[string]interface{}{
			"code":    403,
			"message": "无权限访问",
		})
		return
	}

	// 解析日期参数
	var startDate, endDate *time.Time
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = &t
		}
	}
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = &t
		}
	}

	stats, err := h.statisticsService.GetAuctionStatistics(ctx, startDate, endDate)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取竞拍统计失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, stats)
}

// GetRevenueStatistics 获取收入统计
// @Summary 获取收入统计
// @Description 获取收入相关的统计数据
// @Tags statistics
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "开始日期 (YYYY-MM-DD)"
// @Param end_date query string false "结束日期 (YYYY-MM-DD)"
// @Param category query string false "商品类目"
// @Success 200 {object} dao.RevenueStatistics
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /statistics/revenue [get]
func (h *StatisticsHandler) GetRevenueStatistics(ctx context.Context, c *app.RequestContext) {
	// 权限检查
	role, exists := c.Get("role")
	if !exists || role.(int) != 2 {
		c.JSON(403, map[string]interface{}{
			"code":    403,
			"message": "无权限访问",
		})
		return
	}

	// 解析日期参数
	var startDate, endDate *time.Time
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = &t
		}
	}
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = &t
		}
	}

	category := c.Query("category")

	stats, err := h.statisticsService.GetRevenueStatistics(ctx, startDate, endDate, category)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取收入统计失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, stats)
}

// GetUserStatistics 获取用户统计
// @Summary 获取用户统计
// @Description 获取用户相关的统计数据
// @Tags statistics
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "开始日期 (YYYY-MM-DD)"
// @Param end_date query string false "结束日期 (YYYY-MM-DD)"
// @Success 200 {object} dao.UserStatistics
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /statistics/users [get]
func (h *StatisticsHandler) GetUserStatistics(ctx context.Context, c *app.RequestContext) {
	// 权限检查
	role, exists := c.Get("role")
	if !exists || role.(int) != 2 {
		c.JSON(403, map[string]interface{}{
			"code":    403,
			"message": "无权限访问",
		})
		return
	}

	// 解析日期参数
	var startDate, endDate *time.Time
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = &t
		}
	}
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = &t
		}
	}

	stats, err := h.statisticsService.GetUserStatistics(ctx, startDate, endDate)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取用户统计失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, stats)
}
