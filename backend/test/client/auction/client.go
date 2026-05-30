// Package auction 是 E2E 测试用的业务客户端 SDK。
//
// 设计原则：
//   - 每次 HTTP 调用统一返回 StepResult（含 step 名 / 耗时 / 成功否 / refID / message / err）
//     便于编排器把链路可视化为 StepTimeline。
//   - 所有调用通过 gateway，注入 X-User-ID 头部以兼容 sky-lamp 等不支持 body fallback 的接口。
//   - 出价接口同时把 user_id 塞进 body，作为兜底。
package auction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// StepResult E2E 单步结果（同 spec §M3.2 StepResult）
type StepResult struct {
	Step       string `json:"step"`
	OK         bool   `json:"ok"`
	StatusCode int    `json:"status_code"`
	RefID      int64  `json:"ref_id,omitempty"` // 创建出来的资源 ID
	Message    string `json:"message,omitempty"`
	DurationMs int64  `json:"duration_ms"`
	Err        error  `json:"-"`
}

// Client 业务 HTTP 客户端
type Client struct {
	baseURL string
	hc      *http.Client
}

// NewClient 构造（baseURL 例如 http://localhost:8080）
func NewClient(baseURL string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	tr := &http.Transport{
		MaxIdleConns:        256,
		MaxIdleConnsPerHost: 64,
		IdleConnTimeout:     60 * time.Second,
	}
	return &Client{
		baseURL: baseURL,
		hc:      &http.Client{Transport: tr, Timeout: timeout},
	}
}

// ---------- 请求/响应 DTO ----------

// CreateProductReq 创建拍品
type CreateProductReq struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Images      []string `json:"images,omitempty"`
	Status      int      `json:"status,omitempty"`
}

// CreateAuctionReq 创建拍卖
type CreateAuctionReq struct {
	ProductID  int64   `json:"product_id"`
	StartPrice float64 `json:"start_price"`
	Increment  float64 `json:"increment"`
	Duration   int     `json:"duration"` // 秒
}

// Auction 拍卖快照（仅 E2E/AntiSnipe 关心的字段）
type Auction struct {
	ID           int64     `json:"id"`
	ProductID    int64     `json:"product_id"`
	Status       int       `json:"status"` // 0=Pending 1=Ongoing 2=Delayed 3=Ended 4=Cancelled
	CurrentPrice float64   `json:"current_price"`
	WinnerID     int64     `json:"winner_id"`
	DelayUsed    int       `json:"delay_used"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
}

// Order 订单快照
type Order struct {
	ID         int64   `json:"id"`
	AuctionID  int64   `json:"auction_id"`
	ProductID  int64   `json:"product_id"`
	WinnerID   int64   `json:"winner_id"`
	FinalPrice float64 `json:"final_price"`
	Status     int     `json:"status"`
}

type ordersResp struct {
	Items []Order `json:"items"`
	Total int64   `json:"total"`
}

type skyLampResp struct {
	Code         int `json:"code"`
	Subscription struct {
		ID int64 `json:"id"`
	} `json:"subscription"`
}

// ---------- 各步骤实现 ----------

// CreateProduct POST /api/v1/products
func (c *Client) CreateProduct(ctx context.Context, userID int64, req CreateProductReq) StepResult {
	var resp struct {
		ID int64 `json:"id"`
	}
	step := c.do(ctx, "create_product", http.MethodPost, "/api/v1/products", userID, req, &resp)
	step.RefID = resp.ID
	return step
}

// CreateAuction POST /api/v1/auctions
func (c *Client) CreateAuction(ctx context.Context, userID int64, req CreateAuctionReq) StepResult {
	var resp struct {
		ID int64 `json:"id"`
	}
	step := c.do(ctx, "create_auction", http.MethodPost, "/api/v1/auctions", userID, req, &resp)
	step.RefID = resp.ID
	return step
}

// PlaceBid POST /api/v1/auctions/{id}/bids
func (c *Client) PlaceBid(ctx context.Context, userID, auctionID int64, amount float64) StepResult {
	body := map[string]any{"amount": amount, "user_id": userID}
	path := "/api/v1/auctions/" + strconv.FormatInt(auctionID, 10) + "/bids"
	return c.do(ctx, "bid", http.MethodPost, path, userID, body, nil)
}

// GetAuction GET /api/v1/auctions/{id}
func (c *Client) GetAuction(ctx context.Context, auctionID int64) (Auction, StepResult) {
	var a Auction
	path := "/api/v1/auctions/" + strconv.FormatInt(auctionID, 10)
	step := c.do(ctx, "get_auction", http.MethodGet, path, 0, nil, &a)
	return a, step
}

// WaitAuctionStarted 轮询直到 status >= 1
func (c *Client) WaitAuctionStarted(ctx context.Context, auctionID int64, interval, timeout time.Duration) StepResult {
	return c.poll(ctx, "wait_started", auctionID, interval, timeout, func(a Auction) bool {
		return a.Status >= 1
	})
}

// WaitAuctionEnded 轮询直到 status >= 2
func (c *Client) WaitAuctionEnded(ctx context.Context, auctionID int64, interval, timeout time.Duration) StepResult {
	return c.poll(ctx, "wait_ended", auctionID, interval, timeout, func(a Auction) bool {
		return a.Status >= 2
	})
}

// SubscribeSkyLamp POST /api/v1/sky-lamp/subscriptions
func (c *Client) SubscribeSkyLamp(ctx context.Context, userID, auctionID int64) StepResult {
	body := map[string]any{"auction_id": auctionID}
	var resp skyLampResp
	step := c.do(ctx, "skylamp_subscribe", http.MethodPost, "/api/v1/sky-lamp/subscriptions", userID, body, &resp)
	step.RefID = resp.Subscription.ID
	return step
}

// FindOrdersByAuction 调 /api/v1/orders?user_id=winner，再按 auction_id 客户端过滤
func (c *Client) FindOrdersByAuction(ctx context.Context, winnerID, auctionID int64) ([]Order, StepResult) {
	var resp ordersResp
	path := "/api/v1/orders?user_id=" + strconv.FormatInt(winnerID, 10) + "&page=1&page_size=100"
	step := c.do(ctx, "find_orders", http.MethodGet, path, winnerID, nil, &resp)
	if !step.OK {
		return nil, step
	}
	out := make([]Order, 0, len(resp.Items))
	for _, o := range resp.Items {
		if o.AuctionID == auctionID {
			out = append(out, o)
		}
	}
	return out, step
}

// ---------- 内部工具 ----------

// do 通用 HTTP 调用 + StepResult 装配
func (c *Client) do(ctx context.Context, step, method, path string, userID int64, body any, out any) StepResult {
	start := time.Now()
	res := StepResult{Step: step}

	var rd *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rd)
	if err != nil {
		res.Err = err
		res.Message = err.Error()
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	req.Header.Set("Content-Type", "application/json")
	if userID > 0 {
		// 注入网关身份头，绕过 JWT；auction-service 的 gatewayIdentityMiddleware 会读
		req.Header.Set("X-User-ID", strconv.FormatInt(userID, 10))
		req.Header.Set("X-Username", "test_user_"+strconv.FormatInt(userID, 10))
		req.Header.Set("X-User-Role", "user")
	}

	resp, err := c.hc.Do(req)
	res.DurationMs = time.Since(start).Milliseconds()
	if err != nil {
		res.Err = err
		res.Message = err.Error()
		return res
	}
	defer resp.Body.Close()
	res.StatusCode = resp.StatusCode

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		res.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return res
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			res.Err = err
			res.Message = "decode: " + err.Error()
			return res
		}
	}
	res.OK = true
	return res
}

// poll 轮询拍卖状态
func (c *Client) poll(ctx context.Context, stepName string, auctionID int64, interval, timeout time.Duration, ok func(Auction) bool) StepResult {
	start := time.Now()
	deadline := start.Add(timeout)
	res := StepResult{Step: stepName}
	for {
		a, sub := c.GetAuction(ctx, auctionID)
		if sub.OK && ok(a) {
			res.OK = true
			res.RefID = a.ID
			res.DurationMs = time.Since(start).Milliseconds()
			return res
		}
		if time.Now().After(deadline) {
			res.Message = "timeout waiting"
			res.DurationMs = time.Since(start).Milliseconds()
			return res
		}
		select {
		case <-ctx.Done():
			res.Message = "context cancelled"
			res.Err = ctx.Err()
			res.DurationMs = time.Since(start).Milliseconds()
			return res
		case <-time.After(interval):
		}
	}
}
