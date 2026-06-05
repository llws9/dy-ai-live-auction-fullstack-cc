// Package growthbook 是对官方 GrowthBook Go SDK 的薄封装。
// 设计目标:
//   - 复用官方 SDK 的 hash 分桶/规则匹配/SSE 拉取等核心能力,避免自研客户端的实现陷阱;
//   - 在 SDK 之上叠加 Prometheus 指标 (实验分配/查看/完成);
//   - 提供工具函数:批量评估已知 feature key、序列化为 X-Experiment-Context header 字符串。
package growthbook

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	gb "github.com/growthbook/growthbook-golang"

	"gateway-service/pkg/metrics"
)

// KnownFeatureKeys 网关侧会主动评估并下发到下游/前端的 feature key 列表。
// 新增 feature 时只需在此追加,无需改动 middleware/handler。
var KnownFeatureKeys = []string{
	"new-auction-ui-theme",
	"new-bidding-algorithm",
	"bid-button-color",
	"price-suggestion-strategy",
	"live-start-popup-visibility",
}

// Attributes 业务侧用户属性。仅承载 gateway 层关心的字段,
// 内部通过 ToMap 转换为 GrowthBook 标准属性,避免上层依赖 SDK 类型。
type Attributes struct {
	UserID     string
	Role       int
	Email      string
	DeviceType string
}

// ToMap 将属性序列化为 GrowthBook SDK 所需的 map 格式。
func (a *Attributes) ToMap() map[string]any {
	m := map[string]any{
		"id":   a.UserID,
		"role": a.Role,
	}
	if a.Email != "" {
		m["email"] = a.Email
	}
	if a.DeviceType != "" {
		m["deviceType"] = a.DeviceType
	}
	return m
}

// FeatureSnapshot 网关评估后的 feature 结果摘要。
// 与前后端约定的 X-Experiment-Context JSON 结构保持一致。
type FeatureSnapshot struct {
	On        bool   `json:"on"`
	Value     any    `json:"value,omitempty"`
	Variation string `json:"variation,omitempty"`
}

// Client 对官方 SDK 的薄封装。
type Client struct {
	inner   *gb.Client
	enabled bool
	metrics *metrics.Metrics
}

// NewClient 创建并初始化主客户端。enabled=false 时返回 no-op 客户端。
// 同步等待首次特性加载完成 (有 10s 超时),失败仅打日志不阻塞启动。
func NewClient(ctx context.Context, apiHost, clientKey string, enabled bool, m *metrics.Metrics) (*Client, error) {
	if !enabled {
		log.Println("GrowthBook client disabled by config")
		return &Client{enabled: false, metrics: m}, nil
	}

	inner, err := gb.NewClient(
		ctx,
		gb.WithApiHost(apiHost),
		gb.WithClientKey(clientKey),
		// 5 分钟轮询拉取 feature 配置;若环境支持 SSE 可改为 WithSseDataSource()
		gb.WithPollDataSource(5*time.Minute),
		gb.WithExperimentCallback(func(_ context.Context, exp *gb.Experiment, res *gb.ExperimentResult, _ any) {
			if m == nil || exp == nil || res == nil {
				return
			}
			m.ExperimentAssigned.WithLabelValues(exp.Key, fmt.Sprintf("%d", res.VariationId)).Inc()
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("init growthbook client failed: %w", err)
	}

	loadCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := inner.EnsureLoaded(loadCtx); err != nil {
		log.Printf("Warning: GrowthBook initial features load failed: %v", err)
	} else {
		log.Println("GrowthBook features loaded successfully")
	}

	return &Client{inner: inner, enabled: true, metrics: m}, nil
}

// Close 释放后台数据源/插件,主进程退出时调用。
func (c *Client) Close() error {
	if c.inner != nil {
		return c.inner.Close()
	}
	return nil
}

// Enabled 是否启用 (用于上游短路判断)。
func (c *Client) Enabled() bool {
	return c.enabled
}

// EvalFeatures 批量评估给定 keys,返回 key -> snapshot map。
// disabled 或 attrs 为空时返回空 map (调用方需自行处理"无实验上下文"的降级)。
func (c *Client) EvalFeatures(ctx context.Context, attrs *Attributes, keys []string) map[string]FeatureSnapshot {
	out := make(map[string]FeatureSnapshot)
	if !c.enabled || c.inner == nil || attrs == nil {
		return out
	}

	child, err := c.inner.WithAttributes(gb.Attributes(attrs.ToMap()))
	if err != nil {
		log.Printf("growthbook: build child client failed: %v", err)
		return out
	}

	for _, key := range keys {
		fr := child.EvalFeature(ctx, key)
		if fr == nil {
			continue
		}
		snap := FeatureSnapshot{On: fr.On, Value: fr.Value}
		if fr.ExperimentResult != nil && fr.ExperimentResult.Key != "" {
			snap.Variation = fr.ExperimentResult.Key
		}
		out[key] = snap
	}
	return out
}

// SerializeFeatures 把网关评估的 feature 结果序列化为 X-Experiment-Context 头部字符串。
// 空 map 返回空串,调用方据此决定是否设置 header。
func SerializeFeatures(features map[string]FeatureSnapshot) string {
	if len(features) == 0 {
		return ""
	}
	b, err := json.Marshal(features)
	if err != nil {
		log.Printf("growthbook: serialize features failed: %v", err)
		return ""
	}
	return string(b)
}

// TrackViewed 记录实验曝光。前端 trackingCallback 触发 → 后端落点。
func (c *Client) TrackViewed(experimentKey, variation string) {
	if c.metrics != nil {
		c.metrics.ExperimentViewed.WithLabelValues(experimentKey, variation).Inc()
	}
}

// TrackCompleted 记录实验完成 (转化)。
func (c *Client) TrackCompleted(experimentKey, variation string) {
	if c.metrics != nil {
		c.metrics.ExperimentCompleted.WithLabelValues(experimentKey, variation).Inc()
	}
}
