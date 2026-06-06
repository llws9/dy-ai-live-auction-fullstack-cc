package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

// UserStats 是 GET /api/v1/users/me/stats 的响应数据。
type UserStats struct {
	FollowingCount      int64 `json:"following_count"`
	AuctionHistoryCount int64 `json:"auction_history_count"`
	WonCount            int64 `json:"won_count"`
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
func (f *UserStatsFetcher) Fetch(ctx context.Context, userID int64) (UserStats, error) {
	var (
		out      UserStats
		wg       sync.WaitGroup
		errMu    sync.Mutex
		firstErr error
	)
	recordErr := func(err error) {
		if err == nil {
			return
		}
		errMu.Lock()
		defer errMu.Unlock()
		if firstErr == nil {
			firstErr = err
		}
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		total, err := f.fetchFollowingTotal(ctx, userID)
		if err != nil {
			recordErr(err)
			return
		}
		out.FollowingCount = total
	}()
	go func() {
		defer wg.Done()
		total, won, err := f.fetchOrderHistory(ctx, userID)
		if err != nil {
			recordErr(err)
			return
		}
		out.AuctionHistoryCount = total
		out.WonCount = won
	}()
	wg.Wait()
	if firstErr != nil {
		return UserStats{}, firstErr
	}
	return out, nil
}

// fetchFollowingTotal 调 auction-service `/api/v1/user/followed-live-streams?page_size=1` 拿 total。
func (f *UserStatsFetcher) fetchFollowingTotal(ctx context.Context, userID int64) (int64, error) {
	url := f.auctionURL + "/api/v1/user/followed-live-streams?page=1&page_size=1"
	body, err := f.doGET(ctx, url, userID)
	if err != nil {
		return 0, fmt.Errorf("fetch following total: %w", err)
	}
	// auction-service follow handler 响应：{ "code":200, "data":{ "items":[], "total":N, "page":..., "page_size":... } }
	var wrap struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		return 0, fmt.Errorf("decode following total: %w", err)
	}
	if wrap.Code != 0 && wrap.Code != 200 {
		return 0, fmt.Errorf("following total business code: %d", wrap.Code)
	}
	return wrap.Data.Total, nil
}

// fetchOrderHistory 调 product-service `/api/v1/orders/history`，
// total 即 auction_history_count；won_count 由 items 中 is_winner=true 累计。
//
// 注：product 的 OrderHandler.GetUserHistory 当前是扁平 {items, total, ...} 不带 code 包裹，
// 与 auction follow 不同。这里两种 shape 都处理，以容错下游契约演进。
func (f *UserStatsFetcher) fetchOrderHistory(ctx context.Context, userID int64) (total int64, won int64, err error) {
	// page_size=200 单次拿一页足以覆盖大多数普通用户；超出本期不精确（已在决策中接受）
	url := f.productURL + "/api/v1/orders/history?page=1&page_size=200"
	body, hitErr := f.doGET(ctx, url, userID)
	if hitErr != nil {
		return 0, 0, fmt.Errorf("fetch order history: %w", hitErr)
	}

	// 兼容两种 shape：扁平 / data 包裹
	var flat struct {
		Items []map[string]interface{} `json:"items"`
		List  []map[string]interface{} `json:"list"`
		Total int64                    `json:"total"`
	}
	var wrapped struct {
		Code int `json:"code"`
		Data struct {
			Items []map[string]interface{} `json:"items"`
			List  []map[string]interface{} `json:"list"`
			Total int64                    `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &flat); err == nil && (flat.Items != nil || flat.List != nil) {
		return flat.Total, countWinners(firstNonNilHistoryItems(flat.Items, flat.List)), nil
	}
	if err := json.Unmarshal(body, &wrapped); err == nil {
		return wrapped.Data.Total, countWinners(firstNonNilHistoryItems(wrapped.Data.Items, wrapped.Data.List)), nil
	}
	return 0, 0, fmt.Errorf("decode order history")
}

func firstNonNilHistoryItems(items, list []map[string]interface{}) []map[string]interface{} {
	if items != nil {
		return items
	}
	return list
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

func (f *UserStatsFetcher) doGET(ctx context.Context, url string, userID int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-User-ID", strconv.FormatInt(userID, 10))
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// UserStatsHandler 是 GET /api/v1/users/me/stats 的 HTTP 入口。
//
// 鉴权：authGroup → JWTAuth 已注入 user_id 到 c.Set("user_id", ...)。
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

	stats, err := h.fetcher.Fetch(ctx, uid)
	if err != nil {
		c.JSON(502, map[string]interface{}{
			"code":    502,
			"message": "获取用户统计失败: " + err.Error(),
		})
		return
	}
	c.JSON(200, map[string]interface{}{
		"code": 200,
		"data": stats,
	})
}
