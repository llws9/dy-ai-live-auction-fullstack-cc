package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

// UserStats 是 GET /api/v1/users/me/stats 的响应数据。
//
// 三个字段都用 *int64：单一下游失败时该字段返回 null（前端显示 -），
// 其余字段仍可正常展示。spec A §2.1 / tasks T2.7。
type UserStats struct {
	FollowingCount      *int64 `json:"following_count"`
	AuctionHistoryCount *int64 `json:"auction_history_count"`
	WonCount            *int64 `json:"won_count"`
}

// UserStatsFetcher 并行调下游聚合用户统计。
type UserStatsFetcher struct {
	auctionURL string
	productURL string
	client     *http.Client
}

// NewUserStatsFetcher 构造一个超时受控的聚合器。
func NewUserStatsFetcher(auctionURL, productURL string, timeout time.Duration) *UserStatsFetcher {
	return &UserStatsFetcher{
		auctionURL: auctionURL,
		productURL: productURL,
		client:     &http.Client{Timeout: timeout},
	}
}

// Fetch 同时拉 auction follow count + product orders history，并聚合 won_count。
//
// 任意子调用失败时对应字段保持 nil，不返回 error —— 这是软降级语义，
// 调用方（HTTP handler）总是 200 OK 即可。
func (f *UserStatsFetcher) Fetch(ctx context.Context, userID int64) UserStats {
	var (
		out UserStats
		wg  sync.WaitGroup
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		total, ok := f.fetchFollowingTotal(ctx, userID)
		if ok {
			out.FollowingCount = &total
		}
	}()
	go func() {
		defer wg.Done()
		total, won, ok := f.fetchOrderHistory(ctx, userID)
		if ok {
			out.AuctionHistoryCount = &total
			out.WonCount = &won
		}
	}()
	wg.Wait()
	return out
}

// fetchFollowingTotal 调 auction-service `/api/v1/user/followed-live-streams?page_size=1` 拿 total。
func (f *UserStatsFetcher) fetchFollowingTotal(ctx context.Context, userID int64) (int64, bool) {
	url := f.auctionURL + "/api/v1/user/followed-live-streams?page=1&page_size=1"
	body, ok := f.doGET(ctx, url, userID)
	if !ok {
		return 0, false
	}
	// auction-service follow handler 响应：{ "code":200, "data":{ "items":[], "total":N, "page":..., "page_size":... } }
	var wrap struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		return 0, false
	}
	if wrap.Code != 0 && wrap.Code != 200 {
		return 0, false
	}
	return wrap.Data.Total, true
}

// fetchOrderHistory 调 product-service `/api/v1/orders/history`，
// total 即 auction_history_count；won_count 由 items 中 is_winner=true 累计。
//
// 注：product 的 OrderHandler.GetUserHistory 当前是扁平 {items, total, ...} 不带 code 包裹，
// 与 auction follow 不同。这里两种 shape 都处理，以容错下游契约演进。
func (f *UserStatsFetcher) fetchOrderHistory(ctx context.Context, userID int64) (total int64, won int64, ok bool) {
	// page_size=200 单次拿一页足以覆盖大多数普通用户；超出本期不精确（已在决策中接受）
	url := f.productURL + "/api/v1/orders/history?page=1&page_size=200"
	body, hit := f.doGET(ctx, url, userID)
	if !hit {
		return 0, 0, false
	}

	// 兼容两种 shape：扁平 / data 包裹
	var flat struct {
		Items []map[string]interface{} `json:"items"`
		Total int64                    `json:"total"`
	}
	var wrapped struct {
		Code int `json:"code"`
		Data struct {
			Items []map[string]interface{} `json:"items"`
			Total int64                    `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &flat); err == nil && flat.Items != nil {
		return flat.Total, countWinners(flat.Items), true
	}
	if err := json.Unmarshal(body, &wrapped); err == nil {
		return wrapped.Data.Total, countWinners(wrapped.Data.Items), true
	}
	return 0, 0, false
}

func countWinners(items []map[string]interface{}) int64 {
	var n int64
	for _, it := range items {
		if v, ok := it["is_winner"].(bool); ok && v {
			n++
		}
	}
	return n
}

func (f *UserStatsFetcher) doGET(ctx context.Context, url string, userID int64) ([]byte, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false
	}
	req.Header.Set("X-User-ID", strconv.FormatInt(userID, 10))
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, false
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false
	}
	return body, true
}

// UserStatsHandler 是 GET /api/v1/users/me/stats 的 HTTP 入口。
//
// 鉴权：authGroup → JWTAuth 已注入 user_id 到 c.Set("user_id", ...)。
// 软降级：任意下游失败对应字段返回 null，整体仍 200。
type UserStatsHandler struct {
	fetcher *UserStatsFetcher
}

// NewUserStatsHandler 构造 handler，下游 URL 来自配置。
func NewUserStatsHandler(auctionURL, productURL string, timeout time.Duration) *UserStatsHandler {
	return &UserStatsHandler{
		fetcher: NewUserStatsFetcher(auctionURL, productURL, timeout),
	}
}

// Handle 是 hertz 路由入口。
func (h *UserStatsHandler) Handle(ctx context.Context, c *app.RequestContext) {
	uidVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, map[string]interface{}{"code": 401, "message": "未认证"})
		return
	}
	uid, ok := uidVal.(int64)
	if !ok {
		c.JSON(401, map[string]interface{}{"code": 401, "message": "无效的用户身份"})
		return
	}

	stats := h.fetcher.Fetch(ctx, uid)
	c.JSON(200, map[string]interface{}{
		"code": 200,
		"data": stats,
	})
}
