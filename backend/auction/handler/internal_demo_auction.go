package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"

	"auction-service/dao"
	"auction-service/model"
	"auction-service/websocket"
)

type InternalDemoAuctionHandler struct {
	auctionDAO *dao.AuctionDAO
	timeSync   *websocket.TimeSyncService
}

func NewInternalDemoAuctionHandler(auctionDAO *dao.AuctionDAO, hub *websocket.Hub) *InternalDemoAuctionHandler {
	timeSync := websocket.NewTimeSyncService()
	timeSync.SetHub(hub)
	return &InternalDemoAuctionHandler{auctionDAO: auctionDAO, timeSync: timeSync}
}

func (h *InternalDemoAuctionHandler) Shorten(ctx context.Context, c *app.RequestContext) {
	if h.auctionDAO == nil {
		c.JSON(http.StatusInternalServerError, map[string]any{"code": 500, "message": "auction dao not configured"})
		return
	}

	var req struct {
		AuctionID        int64 `json:"auction_id"`
		RemainingSeconds int   `json:"remaining_seconds"`
	}
	if err := json.Unmarshal(c.Request.Body(), &req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{"code": 400, "message": "invalid json"})
		return
	}
	if req.AuctionID <= 0 {
		c.JSON(http.StatusBadRequest, map[string]any{"code": 400, "message": "invalid auction_id"})
		return
	}
	if req.RemainingSeconds <= 0 || req.RemainingSeconds > 600 {
		c.JSON(http.StatusBadRequest, map[string]any{"code": 400, "message": "remaining_seconds must be between 1 and 600"})
		return
	}

	auction, err := h.auctionDAO.GetByID(ctx, req.AuctionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, map[string]any{"code": 404, "message": "auction not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, map[string]any{"code": 500, "message": "get auction failed"})
		return
	}
	if auction.Status != model.AuctionStatusOngoing && auction.Status != model.AuctionStatusDelayed {
		c.JSON(http.StatusConflict, map[string]any{"code": 409, "message": "auction is not active"})
		return
	}

	newEndTime := time.Now().Add(time.Duration(req.RemainingSeconds) * time.Second)
	if err := h.auctionDAO.UpdateEndTime(ctx, req.AuctionID, newEndTime); err != nil {
		c.JSON(http.StatusInternalServerError, map[string]any{"code": 500, "message": "shorten auction failed"})
		return
	}
	h.timeSync.BroadcastTimeSync(req.AuctionID, newEndTime.UnixMilli())

	c.JSON(http.StatusOK, map[string]any{
		"ok":                true,
		"auction_id":        req.AuctionID,
		"remaining_seconds": req.RemainingSeconds,
		"end_time":          newEndTime.Format(time.RFC3339),
	})
}
