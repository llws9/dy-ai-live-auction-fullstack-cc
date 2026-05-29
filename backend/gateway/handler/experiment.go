package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"gateway-service/middleware"
	"gateway-service/pkg/growthbook"
)

// ExperimentHandler 暴露给前端的 A/B 实验 HTTP 接口。
type ExperimentHandler struct {
	gbClient *growthbook.Client
}

func NewExperimentHandler(gbClient *growthbook.Client) *ExperimentHandler {
	return &ExperimentHandler{gbClient: gbClient}
}

// GetFeatures 返回中间件已预评估的 feature 结果,避免重复评估。
func (h *ExperimentHandler) GetFeatures(_ context.Context, c *app.RequestContext) {
	attrs := middleware.GetExperimentAttributes(c)
	features := middleware.GetExperimentFeatures(c)
	if features == nil {
		features = map[string]growthbook.FeatureSnapshot{}
	}

	resp := map[string]any{
		"features": features,
	}
	if attrs != nil {
		resp["userId"] = attrs.UserID
		resp["userRole"] = attrs.Role
	}

	c.JSON(200, map[string]any{
		"code":    200,
		"message": "success",
		"data":    resp,
	})
}

// TrackViewed 前端 trackingCallback 调用,记录实验曝光。
func (h *ExperimentHandler) TrackViewed(_ context.Context, c *app.RequestContext) {
	var req struct {
		Experiment string `json:"experiment"`
		Variation  string `json:"variation"`
	}
	if err := c.BindJSON(&req); err != nil || req.Experiment == "" {
		c.JSON(400, map[string]any{"code": 400, "message": "参数错误"})
		return
	}
	h.gbClient.TrackViewed(req.Experiment, req.Variation)
	c.JSON(200, map[string]any{"code": 200, "message": "success"})
}

// TrackCompleted 前端转化埋点。
func (h *ExperimentHandler) TrackCompleted(_ context.Context, c *app.RequestContext) {
	var req struct {
		Experiment string `json:"experiment"`
		Variation  string `json:"variation"`
	}
	if err := c.BindJSON(&req); err != nil || req.Experiment == "" {
		c.JSON(400, map[string]any{"code": 400, "message": "参数错误"})
		return
	}
	h.gbClient.TrackCompleted(req.Experiment, req.Variation)
	c.JSON(200, map[string]any{"code": 200, "message": "success"})
}
