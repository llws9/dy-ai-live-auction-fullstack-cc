package handler

import (
	"context"
	"errors"
	"strconv"
	"time"

	"auction-service/service"

	"github.com/cloudwego/hertz/pkg/app"
)

const (
	statisticsRoleAdmin    = "admin"
	statisticsRoleMerchant = "merchant"
)

type StatisticsHandler struct {
	statisticsService *service.StatisticsService
}

func NewStatisticsHandler(statisticsService *service.StatisticsService) *StatisticsHandler {
	return &StatisticsHandler{statisticsService: statisticsService}
}

func (h *StatisticsHandler) GetAuctionStatistics(ctx context.Context, c *app.RequestContext) {
	creatorID, ok := readAuctionStatisticsScope(c)
	if !ok {
		return
	}
	if groupBy := c.Query("group_by"); groupBy != "" && groupBy != "day" {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "group_by only supports day"})
		return
	}

	startDate, endDate, err := parseAuctionStatisticsRange(c, time.Now())
	if err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": err.Error()})
		return
	}
	stats, err := h.statisticsService.GetAuctionDailyStats(ctx, startDate, endDate, creatorID)
	if err != nil {
		if errors.Is(err, service.ErrInvalidStatisticsRange) {
			c.JSON(400, map[string]interface{}{"code": 400, "message": "invalid statistics date range"})
			return
		}
		c.JSON(500, map[string]interface{}{"code": 500, "message": "获取竞拍统计失败: " + err.Error()})
		return
	}
	c.JSON(200, stats)
}

func readAuctionStatisticsScope(c *app.RequestContext) (*int64, bool) {
	switch string(c.GetHeader("X-User-Role")) {
	case statisticsRoleAdmin:
		return nil, true
	case statisticsRoleMerchant:
		userID, err := strconv.ParseInt(string(c.GetHeader("X-User-ID")), 10, 64)
		if err != nil || userID <= 0 {
			c.JSON(401, map[string]interface{}{"code": 401, "message": "未认证，请先登录"})
			return nil, false
		}
		return &userID, true
	default:
		c.JSON(403, map[string]interface{}{"code": 403, "message": "权限不足"})
		return nil, false
	}
}

func parseAuctionStatisticsRange(c *app.RequestContext, now time.Time) (time.Time, time.Time, error) {
	start, end := defaultAuctionStatisticsRange(now)
	if raw := c.Query("start_date"); raw != "" {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		start = parsed
	}
	if raw := c.Query("end_date"); raw != "" {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		end = parsed
	}
	return start, end, nil
}

func defaultAuctionStatisticsRange(now time.Time) (time.Time, time.Time) {
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	start := end.AddDate(0, 0, -6)
	return start, end
}
