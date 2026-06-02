package handler

import (
	"context"
	"errors"
	"log"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"
	"product-service/client"
	"product-service/model"
	"product-service/service"
)

const maxAdminLiveStreamPageSize = 100

const maxPublicLiveStreamPageSize = 50

type LiveStreamHandler struct {
	liveStreamService *service.LiveStreamService
	auctionClient     *client.AuctionClient
}

func NewLiveStreamHandler(liveStreamService *service.LiveStreamService) *LiveStreamHandler {
	return &LiveStreamHandler{
		liveStreamService: liveStreamService,
	}
}

func (h *LiveStreamHandler) SetAuctionClient(ac *client.AuctionClient) {
	h.auctionClient = ac
}

// ListAdmin 管理端直播间列表 (T011)
func (h *LiveStreamHandler) ListAdmin(ctx context.Context, c *app.RequestContext) {
	if !requireAdminRole(c) {
		return
	}

	page, err := parseAdminLiveStreamIntQuery(c, "page", 1)
	if err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "无效的页码"})
		return
	}
	pageSize, err := parseAdminLiveStreamIntQuery(c, "page_size", 20)
	if err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "无效的分页大小"})
		return
	}
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > maxAdminLiveStreamPageSize {
		pageSize = maxAdminLiveStreamPageSize
	}

	var statusFilter *int
	if statusStr := c.Query("status"); statusStr != "" {
		status, err := strconv.Atoi(statusStr)
		if err != nil || !isValidLiveStreamStatus(model.LiveStreamStatus(status)) {
			c.JSON(400, map[string]interface{}{"code": 400, "message": "无效的直播间状态"})
			return
		}
		statusFilter = &status
	}

	liveStreams, total, err := h.liveStreamService.ListAdmin(ctx, page, pageSize, statusFilter)
	if err != nil {
		log.Printf("LiveStream ListAdmin failed: status=%v page=%d pageSize=%d err=%v", statusFilter, page, pageSize, err)
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取直播间列表失败",
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"list":      h.buildAdminList(ctx, liveStreams),
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

func (h *LiveStreamHandler) buildAdminList(ctx context.Context, liveStreams []model.LiveStream) []map[string]interface{} {
	items := make([]map[string]interface{}, 0, len(liveStreams))
	for _, liveStream := range liveStreams {
		auctionCount := int64(0)
		if h.auctionClient != nil {
			if count, err := h.auctionClient.CountAuctionsByLiveStreamID(ctx, liveStream.ID); err == nil {
				auctionCount = count
			}
		}
		items = append(items, h.buildAdminItem(ctx, liveStream, auctionCount))
	}
	return items
}

func (h *LiveStreamHandler) buildAdminItem(ctx context.Context, liveStream model.LiveStream, auctionCount int64) map[string]interface{} {
	streamerName := liveStream.StreamerName
	if streamerName == "" {
		streamerName = liveStream.Name
	}
	return map[string]interface{}{
		"id":              liveStream.ID,
		"name":            liveStream.Name,
		"description":     liveStream.Description,
		"cover_image":     liveStream.CoverImage,
		"status":          liveStream.Status,
		"streamer_id":     liveStream.CreatorID,
		"streamer_name":   streamerName,
		"streamer_avatar": liveStream.StreamerAvatar,
		"viewer_count":    h.liveStreamService.ViewerCount(ctx, liveStream.ID),
		"auction_count":   auctionCount,
		"ban_reason":      liveStream.BanReason,
		"created_at":      liveStream.CreatedAt,
	}
}

func (h *LiveStreamHandler) EndAdmin(ctx context.Context, c *app.RequestContext) {
	if !requireAdminRole(c) {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "无效的直播间ID"})
		return
	}
	liveStream, err := h.liveStreamService.End(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, map[string]interface{}{"code": 404, "message": "直播间不存在"})
			return
		}
		log.Printf("LiveStream EndAdmin failed: id=%d err=%v", id, err)
		c.JSON(500, map[string]interface{}{"code": 500, "message": "结束直播间失败"})
		return
	}
	c.JSON(200, map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"id":     liveStream.ID,
			"status": liveStream.Status,
			"event":  "live_stream_ended",
		},
	})
}

func (h *LiveStreamHandler) BanAdmin(ctx context.Context, c *app.RequestContext) {
	if !requireAdminRole(c) {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "无效的直播间ID"})
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.BindJSON(&req)
	liveStream, err := h.liveStreamService.Ban(ctx, id, req.Reason)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, map[string]interface{}{"code": 404, "message": "直播间不存在"})
			return
		}
		log.Printf("LiveStream BanAdmin failed: id=%d err=%v", id, err)
		c.JSON(500, map[string]interface{}{"code": 500, "message": "封禁直播间失败"})
		return
	}
	c.JSON(200, map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"id":         liveStream.ID,
			"status":     liveStream.Status,
			"ban_reason": liveStream.BanReason,
		},
	})
}

// ListPublic 公开直播间列表 (H5 feed)：仅返回直播中（status=1）的直播间，
// 并为每个直播间补当前竞拍信息 current_auction_id/current_product_id/current_price。
func (h *LiveStreamHandler) ListPublic(ctx context.Context, c *app.RequestContext) {
	page, err := parseAdminLiveStreamIntQuery(c, "page", 1)
	if err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "无效的页码"})
		return
	}
	pageSize, err := parseAdminLiveStreamIntQuery(c, "page_size", 20)
	if err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "无效的分页大小"})
		return
	}
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > maxPublicLiveStreamPageSize {
		pageSize = maxPublicLiveStreamPageSize
	}

	live := int(model.LiveStreamStatusLive)
	statusFilter := &live

	liveStreams, total, err := h.liveStreamService.ListAdmin(ctx, page, pageSize, statusFilter)
	if err != nil {
		log.Printf("LiveStream ListPublic failed: page=%d pageSize=%d err=%v", page, pageSize, err)
		c.JSON(500, map[string]interface{}{"code": 500, "message": "获取直播间列表失败"})
		return
	}

	current := map[int64]client.CurrentAuctionItem{}
	if h.auctionClient != nil && len(liveStreams) > 0 {
		ids := make([]int64, 0, len(liveStreams))
		for _, ls := range liveStreams {
			ids = append(ids, ls.ID)
		}
		if got, err := h.auctionClient.CurrentByLiveStreamIDs(ctx, ids); err != nil {
			log.Printf("LiveStream ListPublic current-by-live-streams failed (degraded): err=%v", err)
		} else {
			current = got
		}
	}

	list := make([]map[string]interface{}, 0, len(liveStreams))
	for _, ls := range liveStreams {
		var currentAuctionID interface{} = nil
		var currentProductID interface{} = nil
		var currentPrice interface{} = nil
		if item, ok := current[ls.ID]; ok {
			currentAuctionID = item.AuctionID
			currentProductID = item.ProductID
			currentPrice = item.CurrentPrice
		}
		list = append(list, map[string]interface{}{
			"id":                 ls.ID,
			"name":               ls.Name,
			"cover_image":        ls.CoverImage,
			"status":             ls.Status,
			"host_name":          ls.StreamerName,
			"host_avatar":        ls.StreamerAvatar,
			"viewer_count":       h.liveStreamService.ViewerCount(ctx, ls.ID),
			"current_auction_id": currentAuctionID,
			"current_product_id": currentProductID,
			"current_price":      currentPrice,
		})
	}

	c.JSON(200, map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"list":      list,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

func parseAdminLiveStreamIntQuery(c *app.RequestContext, key string, defaultValue int) (int, error) {
	raw := c.Query(key)
	if raw == "" {
		return defaultValue, nil
	}
	return strconv.Atoi(raw)
}

func isValidLiveStreamStatus(status model.LiveStreamStatus) bool {
	switch status {
	case model.LiveStreamStatusNotStarted, model.LiveStreamStatusLive, model.LiveStreamStatusEnded, model.LiveStreamStatusBanned:
		return true
	default:
		return false
	}
}

// GetDetail 直播间详情 (T012)
func (h *LiveStreamHandler) GetDetail(ctx context.Context, c *app.RequestContext) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的直播间ID",
		})
		return
	}

	detail, err := h.liveStreamService.GetDetail(ctx, id)
	if err != nil {
		c.JSON(404, map[string]interface{}{
			"code":    404,
			"message": "直播间不存在",
		})
		return
	}

	// 构建响应
	// T2.4(F-B1) MVP：补齐 H5 详情页所需字段；跨服务依赖（host/follow/viewer）本期降级为占位值，
	// 后续 task 将由 auction-service follow-status / hub viewer 接入。
	var videoURL interface{} = nil
	if detail.VideoURL != "" {
		videoURL = detail.VideoURL
	}

	isFollowing := false
	followersCount := int64(0)
	if h.auctionClient != nil {
		if uidStr := string(c.GetHeader("X-User-ID")); uidStr != "" {
			if uid, err := strconv.ParseInt(uidStr, 10, 64); err == nil && uid > 0 {
				if fs, err := h.auctionClient.GetFollowStatus(ctx, uid, id); err == nil {
					isFollowing = fs.IsFollowing
				}
			}
		}
		if fc, err := h.auctionClient.GetFollowersCount(ctx, id); err == nil {
			followersCount = fc
		}
	}

	result := map[string]interface{}{
		"id":              detail.ID,
		"name":            detail.Name,
		"description":     detail.Description,
		"cover_image":     detail.CoverImage,
		"status":          detail.Status,
		"creator_id":      detail.CreatorID,
		"streamer_id":     detail.CreatorID,
		"streamer_name":   detail.StreamerName,
		"streamer_avatar": detail.StreamerAvatar,
		"created_at":      detail.CreatedAt,
		"host_name":       "",
		"host_avatar":     "",
		"viewer_count":    h.liveStreamService.ViewerCount(ctx, detail.ID),
		"auction_count":   0,
		"ban_reason":      detail.BanReason,
		"video_url":       videoURL,
		"is_following":    isFollowing,
		"followers_count": followersCount,
	}

	c.JSON(200, map[string]interface{}{
		"code": 200,
		"data": result,
	})
}
