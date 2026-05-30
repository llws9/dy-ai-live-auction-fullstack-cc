package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

// InternalUserHandler 暴露 /internal/users/batch 内部接口（T3.3 / spec B §4.1），
// 仅供同 VPC 的其它服务（product-service）调用，由 InternalAuthMiddleware 鉴权。
type InternalUserHandler struct {
	provider UserBatchProvider
}

func NewInternalUserHandler(provider UserBatchProvider) *InternalUserHandler {
	return &InternalUserHandler{provider: provider}
}

// internalUserBatchRequest 是请求体。
type internalUserBatchRequest struct {
	IDs []int64 `json:"ids"`
}

// BatchByIDs 处理 POST /internal/users/batch。
func (h *InternalUserHandler) BatchByIDs(ctx context.Context, c *app.RequestContext) {
	var req internalUserBatchRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误: " + err.Error()})
		return
	}
	resp, err := BuildUserSummaries(ctx, h.provider, req.IDs)
	if err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "success",
		"data":    resp,
	})
}
