package handler

import (
	"context"
	"log"

	"github.com/cloudwego/hertz/pkg/app"

	"auction-service/dao"
	"auction-service/model"
)

type ProductAuctionState struct {
	ProductID           int64  `json:"product_id"`
	ActiveAuctionID     *int64 `json:"active_auction_id,omitempty"`
	ActiveStatus        *int   `json:"active_status,omitempty"`
	LatestAuctionID     *int64 `json:"latest_auction_id,omitempty"`
	LatestAuctionStatus *int   `json:"latest_auction_status,omitempty"`
	LatestAuctionResult string `json:"latest_auction_result,omitempty"`
}

type internalProductAuctionsRequest struct {
	ProductIDs []int64 `json:"product_ids"`
}

type InternalProductAuctionsHandler struct {
	dao *dao.AuctionDAO
}

func NewInternalProductAuctionsHandler(auctionDAO *dao.AuctionDAO) *InternalProductAuctionsHandler {
	return &InternalProductAuctionsHandler{dao: auctionDAO}
}

func (h *InternalProductAuctionsHandler) Handle(ctx context.Context, c *app.RequestContext) {
	var req internalProductAuctionsRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误: " + err.Error()})
		return
	}

	items, err := h.buildStates(ctx, req.ProductIDs)
	if err != nil {
		log.Printf("internal product auction states failed: product_ids=%v err=%v", req.ProductIDs, err)
		c.JSON(500, map[string]interface{}{"code": 500, "message": "internal error"})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "success",
		"data":    map[string]interface{}{"items": items},
	})
}

func (h *InternalProductAuctionsHandler) buildStates(ctx context.Context, productIDs []int64) ([]ProductAuctionState, error) {
	if len(productIDs) == 0 {
		return []ProductAuctionState{}, nil
	}
	seen := make(map[int64]struct{}, len(productIDs))
	items := make([]ProductAuctionState, 0, len(productIDs))
	for _, productID := range productIDs {
		if productID <= 0 {
			continue
		}
		if _, ok := seen[productID]; ok {
			continue
		}
		seen[productID] = struct{}{}

		item := ProductAuctionState{ProductID: productID}
		active, err := h.dao.GetActiveByProductID(ctx, productID)
		if err != nil {
			return nil, err
		}
		if active != nil {
			status := int(active.Status)
			item.ActiveAuctionID = &active.ID
			item.ActiveStatus = &status
		}

		latest, err := h.dao.GetLatestTerminalByProductID(ctx, productID)
		if err != nil {
			return nil, err
		}
		if latest != nil {
			status := int(latest.Status)
			item.LatestAuctionID = &latest.ID
			item.LatestAuctionStatus = &status
			if latest.Status == model.AuctionStatusEnded && latest.WinnerID != nil {
				item.LatestAuctionResult = "sold"
			}
			if latest.Status == model.AuctionStatusEnded && latest.WinnerID == nil {
				item.LatestAuctionResult = "unsold"
			}
		}

		if item.ActiveAuctionID != nil || item.LatestAuctionID != nil {
			items = append(items, item)
		}
	}
	return items, nil
}
