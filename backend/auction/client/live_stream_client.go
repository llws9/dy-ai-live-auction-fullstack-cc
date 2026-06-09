// Package client 包含 LiveStreamClient：auction-service 调 product-service 的
// /internal/live-streams/batch（spec B §4.1）。
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// LiveStreamSummary 是 product-service 内部接口返回的直播间摘要。
// 字段与 spec B §4.1 内部接口契约一致。
type LiveStreamSummary struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	CoverImage  string `json:"cover_image"`
	Status      int    `json:"status"`
	CreatorID   int64  `json:"creator_id"`
	ViewerCount int64  `json:"viewer_count"`
}

// LiveStreamClient 抽象 auction-service 对 product-service 直播间内部接口的依赖。
type LiveStreamClient interface {
	// BatchGetLiveStreams 调 POST /internal/live-streams/batch，按 id 批量取摘要。
	// 当 ids 为空时不发起 HTTP 调用，直接返回空 map。
	BatchGetLiveStreams(ctx context.Context, ids []int64) (map[int64]LiveStreamSummary, error)
}

// HTTPLiveStreamClient 基于 net/http 的实现。
type HTTPLiveStreamClient struct {
	baseURL       string
	hc            *http.Client
	internalToken string
}

// NewHTTPLiveStreamClient 构造客户端。baseURL 形如 "http://product-service:8081"。
func NewHTTPLiveStreamClient(baseURL string, timeout time.Duration) *HTTPLiveStreamClient {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &HTTPLiveStreamClient{
		baseURL: baseURL,
		hc:      &http.Client{Timeout: timeout},
	}
}

// SetInternalToken 注入服务间鉴权 token。
func (c *HTTPLiveStreamClient) SetInternalToken(token string) {
	c.internalToken = token
}

type liveStreamBatchResponse struct {
	Code int `json:"code"`
	Data struct {
		Items []LiveStreamSummary `json:"items"`
	} `json:"data"`
	Message string `json:"message"`
}

// BatchGetLiveStreams 实现见接口注释。
func (c *HTTPLiveStreamClient) BatchGetLiveStreams(ctx context.Context, ids []int64) (map[int64]LiveStreamSummary, error) {
	if len(ids) == 0 {
		return map[int64]LiveStreamSummary{}, nil
	}
	payload, err := json.Marshal(map[string]interface{}{"ids": ids})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	endpoint := c.baseURL + "/internal/live-streams/batch"
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
	var body liveStreamBatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	out := make(map[int64]LiveStreamSummary, len(body.Data.Items))
	for _, it := range body.Data.Items {
		out[it.ID] = it
	}
	return out, nil
}
