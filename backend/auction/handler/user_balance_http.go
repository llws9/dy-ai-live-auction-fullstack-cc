package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/shopspring/decimal"
)

// UserBalanceHandler GET /api/v1/user/balance（T3.1 / spec A F-A2）。
//
// user_id 由 gateway JWTAuth 通过 X-User-ID header 注入到 c.Set("user_id", ...)。
// 编排逻辑在 BuildUserBalanceResponse，handler 只负责 HTTP 解析/序列化。
type UserBalanceHandler struct {
	provider BalanceProvider
	topUpper BalanceTopUpper
}

func NewUserBalanceHandler(store BalanceStore) *UserBalanceHandler {
	return &UserBalanceHandler{provider: store, topUpper: store}
}

func (h *UserBalanceHandler) GetUserBalanceHandler(ctx context.Context, c *app.RequestContext) {
	userID := c.GetInt64("user_id")

	resp, err := BuildUserBalanceResponse(ctx, h.provider, userID)
	if err != nil {
		if err.Error() == "invalid user_id" {
			c.JSON(401, map[string]interface{}{"code": 401, "message": "未登录或无效用户"})
			return
		}
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "查询余额失败",
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code": 200,
		"data": resp,
	})
}

func (h *UserBalanceHandler) TopUpInternal(ctx context.Context, c *app.RequestContext) {
	if h.topUpper == nil {
		c.JSON(http.StatusInternalServerError, map[string]any{
			"code":    500,
			"message": "balance top-up provider not configured",
		})
		return
	}

	var req struct {
		UserID int64  `json:"user_id"`
		Amount string `json:"amount"`
	}
	if err := json.Unmarshal(c.Request.Body(), &req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{
			"code":    400,
			"message": "invalid json",
		})
		return
	}
	if req.UserID <= 0 {
		c.JSON(http.StatusBadRequest, map[string]any{
			"code":    400,
			"message": "invalid user_id",
		})
		return
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{
			"code":    400,
			"message": "invalid amount",
		})
		return
	}
	if !amount.IsPositive() {
		c.JSON(http.StatusBadRequest, map[string]any{
			"code":    400,
			"message": "amount must be positive",
		})
		return
	}

	balance, err := h.topUpper.AddAmount(ctx, req.UserID, amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]any{
			"code":    500,
			"message": "top up balance failed",
		})
		return
	}
	c.JSON(http.StatusOK, map[string]any{
		"code":    0,
		"message": "success",
		"data": map[string]any{
			"user_id": req.UserID,
			"balance": balance.StringFixed(2),
		},
	})
}
