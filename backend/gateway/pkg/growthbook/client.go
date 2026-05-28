package growthbook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"gateway-service/pkg/metrics"
)

// Client GrowthBook SDK 客户端
type Client struct {
	apiHost   string
	clientKey string
	secretKey string
	enabled   bool

	// 缓存 feature flags
	features     map[string]*Feature
	featuresLock sync.RWMutex
	lastRefresh  time.Time

	// HTTP client
	httpClient *http.Client

	// Prometheus metrics
	metrics *metrics.Metrics
}

// Feature 特性开关定义
type Feature struct {
	Key         string
DefaultValue interface{}
	Rules       []FeatureRule
}

// FeatureRule 特性规则
type FeatureRule struct {
	Variation string
	Value     interface{}
	Condition *Condition
}

// Condition 条件定义
type Condition struct {
	Attributes map[string]interface{}
}

// Attributes 用户属性
type Attributes struct {
	ID         string
	Role       int
	Email      string
	DeviceType string
	Browser    string
	Custom     map[string]interface{}
}

// NewClient 创建 GrowthBook 客户端
func NewClient(apiHost, clientKey, secretKey string, enabled bool, m *metrics.Metrics) *Client {
	return &Client{
		apiHost:   apiHost,
		clientKey: clientKey,
		secretKey: secretKey,
		enabled:   enabled,
		features:  make(map[string]*Feature),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		metrics: m,
	}
}

// RefreshFeatures 从 GrowthBook API 获取特性配置
func (c *Client) RefreshFeatures(ctx context.Context) error {
	if !c.enabled {
		return nil
	}

	url := fmt.Sprintf("%s/api/features/%s", c.apiHost, c.clientKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch features failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response failed: %w", err)
	}

	var result struct {
		Features map[string]*Feature `json:"features"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parse response failed: %w", err)
	}

	c.featuresLock.Lock()
	c.features = result.Features
	c.lastRefresh = time.Now()
	c.featuresLock.Unlock()

	return nil
}

// IsOn 检查特性是否开启
func (c *Client) IsOn(featureKey string, attrs *Attributes) bool {
	if !c.enabled {
		return false
	}

	result := c.EvalFeature(featureKey, attrs)
	return result.On
}

// GetValue 获取特性值
func (c *Client) GetValue(featureKey string, attrs *Attributes) interface{} {
	if !c.enabled {
		return nil
	}

	result := c.EvalFeature(featureKey, attrs)
	return result.Value
}

// EvalResult 特性评估结果
type EvalResult struct {
	On       bool
	Value    interface{}
	Variation string
}

// EvalFeature 评估特性开关
func (c *Client) EvalFeature(featureKey string, attrs *Attributes) *EvalResult {
	if !c.enabled {
		return &EvalResult{On: false, Value: nil}
	}

	c.featuresLock.RLock()
	feature, exists := c.features[featureKey]
	c.featuresLock.RUnlock()

	if !exists {
		return &EvalResult{On: false, Value: nil}
	}

	// 评估规则
	for _, rule := range feature.Rules {
		if c.matchesCondition(rule.Condition, attrs) {
			result := &EvalResult{
				Variation: rule.Variation,
				Value:     rule.Value,
			}
			result.On = c.isTruthy(result.Value)

			// 记录实验分配
			if c.metrics != nil {
				c.metrics.ExperimentAssigned.WithLabelValues(
					featureKey,
					result.Variation,
				).Inc()
			}

			return result
		}
	}

	// 使用默认值
	result := &EvalResult{
		Value: feature.DefaultValue,
	}
	result.On = c.isTruthy(result.Value)
	return result
}

// matchesCondition 检查条件是否匹配
func (c *Client) matchesCondition(condition *Condition, attrs *Attributes) bool {
	if condition == nil || condition.Attributes == nil {
		return true // 无条件则匹配所有用户
	}

	// 简化的条件匹配逻辑
	for key, value := range condition.Attributes {
		switch key {
		case "id":
			if attrs.ID != fmt.Sprintf("%v", value) {
				return false
			}
		case "role":
			if attrs.Role != int(value.(float64)) {
				return false
			}
		}
	}

	return true
}

// isTruthy 检查值是否为真
func (c *Client) isTruthy(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case int, int64, float64:
		return v != 0
	case string:
		return v != "" && v != "false" && v != "0"
	default:
		return value != nil
	}
}

// TrackViewed 记录实验查看
func (c *Client) TrackViewed(experimentKey, variation string) {
	if c.metrics != nil {
		c.metrics.ExperimentViewed.WithLabelValues(experimentKey, variation).Inc()
	}
}

// TrackCompleted 记录实验完成
func (c *Client) TrackCompleted(experimentKey, variation string) {
	if c.metrics != nil {
		c.metrics.ExperimentCompleted.WithLabelValues(experimentKey, variation).Inc()
	}
}

// StartRefreshLoop 启动定时刷新循环
func (c *Client) StartRefreshLoop(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.RefreshFeatures(ctx); err != nil {
					// 静默处理错误，不影响服务运行
				}
			}
		}
	}()
}