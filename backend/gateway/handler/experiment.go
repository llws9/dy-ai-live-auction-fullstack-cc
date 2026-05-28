package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"gateway-service/pkg/growthbook"
	"gateway-service/middleware"
)

// ExperimentHandler 实验处理器
type ExperimentHandler struct {
	gbClient *growthbook.Client
}

// NewExperimentHandler 创建实验处理器
func NewExperimentHandler(gbClient *growthbook.Client) *ExperimentHandler {
	return &ExperimentHandler{
		gbClient: gbClient,
	}
}

// GetFeatures 获取用户可用的特性开关
func (h *ExperimentHandler) GetFeatures(ctx context.Context, c *app.RequestContext) {
	attrs := middleware.GetExperimentAttributes(c)

	if attrs == nil {
		c.JSON(200, map[string]interface{}{
			"code":    200,
			"message": "success",
			"data": map[string]interface{}{
				"features": map[string]interface{}{},
			},
		})
		return
	}

	// 获取预定义的特性列表
	featureKeys := []string{
		"new-auction-ui-theme",
		"new-bidding-algorithm",
		"bid-button-color",
		"price-suggestion-strategy",
	}

	features := make(map[string]interface{})
	for _, key := range featureKeys {
		result := h.gbClient.EvalFeature(key, attrs)
		if result.Value != nil {
			features[key] = map[string]interface{}{
				"on":        result.On,
				"value":     result.Value,
				"variation": result.Variation,
			}
		}
	}

	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "success",
		"data": map[string]interface{}{
			"features":  features,
			"userId":    attrs.ID,
			"userRole":  attrs.Role,
		},
	})
}

// TrackViewed 记录实验查看
func (h *ExperimentHandler) TrackViewed(ctx context.Context, c *app.RequestContext) {
	var req struct {
		Experiment string `json:"experiment"`
		Variation  string `json:"variation"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "参数错误",
		})
		return
	}

	h.gbClient.TrackViewed(req.Experiment, req.Variation)

	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "success",
	})
}

// TrackCompleted 记录实验完成
func (h *ExperimentHandler) TrackCompleted(ctx context.Context, c *app.RequestContext) {
	var req struct {
		Experiment string `json:"experiment"`
		Variation  string `json:"variation"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "参数错误",
		})
		return
	}

	h.gbClient.TrackCompleted(req.Experiment, req.Variation)

	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "success",
	})
}