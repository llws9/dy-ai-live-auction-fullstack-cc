package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

// UserBalanceHandler GET /api/v1/user/balance（T3.1 / spec A F-A2）。
//
// user_id 由 gateway JWTAuth 通过 X-User-ID header 注入到 c.Set("user_id", ...)。
// 编排逻辑在 BuildUserBalanceResponse，handler 只负责 HTTP 解析/序列化。
type UserBalanceHandler struct {
	provider BalanceProvider
}

func NewUserBalanceHandler(provider BalanceProvider) *UserBalanceHandler {
	return &UserBalanceHandler{provider: provider}
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
