package handler

import (
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"

	"product-service/service"
)

// CopywritingServiceAPI 抽象 service 行为，便于 handler 测试替身。
type CopywritingServiceAPI interface {
	Generate(ctx context.Context, userID int64, req *service.CopywritingRequest) (*service.CopywritingResponse, error)
}

// CopywritingHandler HTTP 入口。
type CopywritingHandler struct {
	svc CopywritingServiceAPI
}

// NewCopywritingHandler 构造 handler。
func NewCopywritingHandler(svc CopywritingServiceAPI) *CopywritingHandler {
	return &CopywritingHandler{svc: svc}
}

// Generate POST /api/v1/products/ai/copywriting。
func (h *CopywritingHandler) Generate(ctx context.Context, c *app.RequestContext) {
	role := c.GetInt("user_role")
	if role != 1 && role != 2 {
		c.JSON(403, map[string]interface{}{"code": "forbidden_role", "message": "需要商家或管理员权限"})
		return
	}
	userID := c.GetInt64("user_id")
	if userID == 0 {
		c.JSON(401, map[string]interface{}{"code": "unauthorized", "message": "未登录"})
		return
	}

	var req service.CopywritingRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": "invalid_request", "message": err.Error()})
		return
	}

	resp, err := h.svc.Generate(ctx, userID, &req)
	if err != nil {
		mapCopywritingError(c, err)
		return
	}
	c.JSON(200, resp)
}

func mapCopywritingError(c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidRequest):
		c.JSON(400, map[string]interface{}{"code": "invalid_request", "message": err.Error()})
	case errors.Is(err, service.ErrRateLimited):
		c.JSON(429, map[string]interface{}{"code": "rate_limited", "message": err.Error()})
	case errors.Is(err, service.ErrUpstreamTimeout):
		c.JSON(504, map[string]interface{}{"code": "upstream_timeout", "message": err.Error()})
	case errors.Is(err, service.ErrInvalidOutput):
		c.JSON(502, map[string]interface{}{"code": "upstream_invalid_output", "message": err.Error()})
	case errors.Is(err, service.ErrUpstreamFailed):
		c.JSON(502, map[string]interface{}{"code": "upstream_failed", "message": err.Error()})
	default:
		c.JSON(500, map[string]interface{}{"code": "internal_error", "message": err.Error()})
	}
}
