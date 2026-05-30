// Package client 提供调用其它内部服务（如 product-service）的 HTTP 客户端。
//
// 设计取舍：
//   - 仅消费 spec C §5.1 定义的 product-service 内部接口（/internal/products?category_id=、
//     /internal/products/batch），不直接访问 products 表，保持服务边界清晰。
//   - 任何下游错误都向上 bubble，由调用方决定降级策略。本期 T2.2 用户决策：
//     失败时整个 list 接口 5xx（无静默降级）。
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// ProductSummary 是 product-service 内部接口返回的商品摘要。
// 字段与 spec C §5.1.1 / §5.1.2 一致。
type ProductSummary struct {
	ID         int64    `json:"id"`
	Name       string   `json:"name"`
	Images     []string `json:"images"`
	CategoryID *int64   `json:"category_id"`
}

// ProductClient 抽象 auction-service 对 product-service 的依赖，便于测试替换。
type ProductClient interface {
	// ListProductIDsByCategory 调 GET /internal/products?category_id=
	// 返回指定分类下全部商品的 ID 列表（page_size=500 一次取齐，超出再分页）。
	ListProductIDsByCategory(ctx context.Context, categoryID int64) ([]int64, error)

	// BatchGetSummaries 调 POST /internal/products/batch，按 id 批量取摘要。
	// 返回 map[id]ProductSummary 便于按 product_id 回填。
	// 当 ids 为空时不发起 HTTP 调用，直接返回空 map。
	BatchGetSummaries(ctx context.Context, ids []int64) (map[int64]ProductSummary, error)
}

// HTTPProductClient 基于 net/http 的实现。
type HTTPProductClient struct {
	baseURL       string
	hc            *http.Client
	internalToken string
}

// NewHTTPProductClient 构造一个 HTTP 客户端。baseURL 形如 "http://product-service:8081"。
// internalToken 来自 INTERNAL_API_TOKEN 环境变量，用于 X-Internal-Token 鉴权（spec B §4.1）。
func NewHTTPProductClient(baseURL string, timeout time.Duration) *HTTPProductClient {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &HTTPProductClient{
		baseURL: baseURL,
		hc:      &http.Client{Timeout: timeout},
	}
}

// SetInternalToken 注入服务间鉴权 token。
func (c *HTTPProductClient) SetInternalToken(token string) {
	c.internalToken = token
}

// internalListResponse 对应 product-service /internal/products 的响应结构。
type internalListResponse struct {
	Code int `json:"code"`
	Data struct {
		Items []ProductSummary `json:"items"`
		Total int64            `json:"total"`
	} `json:"data"`
	Message string `json:"message"`
}

// internalBatchResponse 对应 /internal/products/batch 的响应结构。
type internalBatchResponse struct {
	Code int `json:"code"`
	Data struct {
		Items []ProductSummary `json:"items"`
	} `json:"data"`
	Message string `json:"message"`
}

// ListProductIDsByCategory 实现见接口注释。
func (c *HTTPProductClient) ListProductIDsByCategory(ctx context.Context, categoryID int64) ([]int64, error) {
	q := url.Values{}
	q.Set("category_id", strconv.FormatInt(categoryID, 10))
	q.Set("page", "1")
	q.Set("page_size", "500")
	endpoint := c.baseURL + "/internal/products?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if c.internalToken != "" {
		req.Header.Set("X-Internal-Token", c.internalToken)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call product-service: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("product-service returned status %d", resp.StatusCode)
	}
	var body internalListResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	ids := make([]int64, 0, len(body.Data.Items))
	for _, it := range body.Data.Items {
		ids = append(ids, it.ID)
	}
	return ids, nil
}

// BatchGetSummaries 实现见接口注释。
func (c *HTTPProductClient) BatchGetSummaries(ctx context.Context, ids []int64) (map[int64]ProductSummary, error) {
	if len(ids) == 0 {
		return map[int64]ProductSummary{}, nil
	}
	payload, err := json.Marshal(map[string]interface{}{"ids": ids})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	endpoint := c.baseURL + "/internal/products/batch"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.internalToken != "" {
		req.Header.Set("X-Internal-Token", c.internalToken)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call product-service: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("product-service returned status %d", resp.StatusCode)
	}
	var body internalBatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	out := make(map[int64]ProductSummary, len(body.Data.Items))
	for _, it := range body.Data.Items {
		out[it.ID] = it
	}
	return out, nil
}
