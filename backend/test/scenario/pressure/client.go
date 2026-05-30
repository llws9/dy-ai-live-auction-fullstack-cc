package pressure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Result 单次请求的结果
type Result struct {
	OK         bool          // 业务+HTTP 都成功才为 true
	StatusCode int           // HTTP 失败时为 HTTP 码；业务失败时为业务 code
	Latency    time.Duration // 端到端耗时
	Err        error         // 网络/超时错误
}

// Client 调 gateway 出价接口的轻量 HTTP 客户端
//   - baseURL：例如 "http://localhost:8080"
//   - authHeader：完整的 Authorization 头值，如 "Bearer xxx"，空则不注入
//   - 复用 *http.Transport，避免连接耗尽
type Client struct {
	baseURL    string
	authHeader string
	hc         *http.Client
}

// NewClient 构造
func NewClient(baseURL, authHeader string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	tr := &http.Transport{
		MaxIdleConns:        2048,
		MaxIdleConnsPerHost: 1024,
		MaxConnsPerHost:     2048,
		IdleConnTimeout:     90 * time.Second,
	}
	return &Client{
		baseURL:    baseURL,
		authHeader: authHeader,
		hc:         &http.Client{Transport: tr, Timeout: timeout},
	}
}

// bidResp 业务响应（统一 code 字段）
type bidResp struct {
	Code int `json:"code"`
}

// PlaceBid 调 gateway POST /api/v1/auctions/{id}/bids
//   amount: 出价金额；auctionID: 拍卖 ID；userID: 出价用户 ID（测试模式）
func (c *Client) PlaceBid(ctx context.Context, amount float64, auctionID, userID int64) Result {
	payload := map[string]any{
		"amount":  amount,
		"user_id": userID,
	}
	body, _ := json.Marshal(payload)

	url := c.baseURL + "/api/v1/auctions/" + strconv.FormatInt(auctionID, 10) + "/bids"
	// 单元测试中 baseURL 已是 httptest.URL，直接拼接也兼容
	if c.baseURL != "" && c.baseURL[len(c.baseURL)-1] == '/' {
		url = c.baseURL + "api/v1/auctions/" + strconv.FormatInt(auctionID, 10) + "/bids"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return Result{OK: false, Err: err}
	}
	req.Header.Set("Content-Type", "application/json")
	if c.authHeader != "" {
		req.Header.Set("Authorization", c.authHeader)
	}

	start := time.Now()
	resp, err := c.hc.Do(req)
	latency := time.Since(start)
	if err != nil {
		return Result{OK: false, Latency: latency, Err: err}
	}
	defer resp.Body.Close()

	// HTTP 错误码：直接返回失败
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{OK: false, StatusCode: resp.StatusCode, Latency: latency}
	}

	// 业务码：code ∈ {0, 200} 视为成功（与 frontend api.ts 约定一致）
	var br bidResp
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&br); err != nil {
		// 响应不可解析 → 视为成功（部分接口可能没 code 字段）
		return Result{OK: true, StatusCode: resp.StatusCode, Latency: latency}
	}
	if br.Code == 0 || br.Code == 200 {
		return Result{OK: true, StatusCode: resp.StatusCode, Latency: latency}
	}
	return Result{
		OK:         false,
		StatusCode: br.Code,
		Latency:    latency,
		Err:        fmt.Errorf("biz_code=%d", br.Code),
	}
}
