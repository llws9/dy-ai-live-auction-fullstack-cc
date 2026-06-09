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
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/shopspring/decimal"
)

const (
	RoleUser     = "user"
	RoleMerchant = "merchant"

	defaultFixtureProductImage = "/assets/default-auction-cover.svg"
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

// Actor 描述一次业务请求使用的测试身份。
type Actor struct {
	UserID   int64
	Username string
	Role     string // user | merchant
}

// Client 业务 HTTP 客户端
type Client struct {
	baseURL       string
	hc            *http.Client
	jwtSecret     string
	internalToken string
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

func (c *Client) SetJWTSecret(secret string) {
	c.jwtSecret = secret
}

func (c *Client) SetInternalToken(token string) {
	c.internalToken = token
}

// ---------- 请求/响应 DTO ----------

// CreateProductReq 创建拍品
type CreateProductReq struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Images      []string `json:"images,omitempty"`
	CategoryID  *int64   `json:"category_id,omitempty"`
	Status      int      `json:"status,omitempty"`
}

const DefaultFixtureProductCategoryID int64 = 12 // ART

func fixtureCategoryID(id int64) *int64 {
	return &id
}

// CreateAuctionReq 创建拍卖
type CreateAuctionReq struct {
	ProductID    int64      `json:"product_id"`
	LiveStreamID int64      `json:"live_stream_id,omitempty"`
	StartPrice   float64    `json:"start_price"`
	Increment    float64    `json:"increment"`
	Duration     int        `json:"duration"` // 秒
	StartTime    *time.Time `json:"start_time,omitempty"`
}

type CreateAuctionRuleReq struct {
	StartPrice         float64 `json:"start_price"`
	Increment          float64 `json:"increment"`
	CapPrice           float64 `json:"cap_price,omitempty"`
	Duration           int     `json:"duration"`
	DelayDuration      int     `json:"delay_duration,omitempty"`
	MaxDelayTime       int     `json:"max_delay_time,omitempty"`
	TriggerDelayBefore int     `json:"trigger_delay_before,omitempty"`
}

type CreateLiveStreamReq struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ProductID   int64  `json:"product_id,omitempty"`
	CoverImage  string `json:"cover_image,omitempty"`
}

type CreateFixedPriceItemReq struct {
	AuctionID    int64  `json:"auction_id"`
	LiveStreamID int64  `json:"live_stream_id"`
	ProductID    int64  `json:"product_id"`
	Price        string `json:"price"`
	Stock        int64  `json:"total_stock"`
}

type LiveStream struct {
	ID     int64 `json:"id"`
	Status any   `json:"status"`
}

type CurrentAuctionItem struct {
	LiveStreamID int64  `json:"live_stream_id"`
	AuctionID    int64  `json:"auction_id"`
	ProductID    int64  `json:"product_id"`
	CurrentPrice string `json:"current_price"`
	Status       int    `json:"status"`
}

type FixedPriceItem struct {
	ID             int64 `json:"id"`
	AuctionID      int64 `json:"auction_id"`
	LiveStreamID   int64 `json:"live_stream_id"`
	Stock          int64 `json:"stock"`
	RemainingStock int64 `json:"remaining_stock"`
}

type FixedPricePurchase struct {
	ID      int64 `json:"id"`
	ItemID  int64 `json:"item_id"`
	OrderID int64 `json:"order_id"`
}

// Auction 拍卖快照（仅 E2E/AntiSnipe 关心的字段）
type Auction struct {
	ID           int64         `json:"id"`
	ProductID    int64         `json:"product_id"`
	Status       int           `json:"status"` // 0=Pending 1=Ongoing 2=Delayed 3=Ended 4=Cancelled
	CurrentPrice float64       `json:"current_price"`
	WinnerID     int64         `json:"winner_id"`
	DelayUsed    int           `json:"delay_used"`
	Rules        *AuctionRules `json:"rules,omitempty"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
}

type AuctionRules struct {
	StartPrice decimal.Decimal `json:"start_price"`
	Increment  decimal.Decimal `json:"increment"`
	CapPrice   decimal.Decimal `json:"cap_price"`
}

type AuctionResult struct {
	AuctionID  int64   `json:"auction_id"`
	ProductID  int64   `json:"product_id"`
	Status     int     `json:"status"`
	FinalPrice float64 `json:"final_price"`
	WinnerID   int64   `json:"winner_id"`
	WonBid     float64 `json:"won_bid"`
}

func (r *AuctionResult) UnmarshalJSON(data []byte) error {
	var aux struct {
		AuctionID  int64           `json:"auction_id"`
		ProductID  int64           `json:"product_id"`
		Status     int             `json:"status"`
		FinalPrice float64         `json:"final_price"`
		WinnerID   int64           `json:"winner_id"`
		WonBid     json.RawMessage `json:"won_bid"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	r.AuctionID = aux.AuctionID
	r.ProductID = aux.ProductID
	r.Status = aux.Status
	r.FinalPrice = aux.FinalPrice
	r.WinnerID = aux.WinnerID
	r.WonBid = 0
	if len(aux.WonBid) == 0 || string(aux.WonBid) == "null" {
		return nil
	}
	if amount, ok := parseJSONFloat(aux.WonBid); ok {
		r.WonBid = amount
		return nil
	}
	if amount, ok := parseWonBidAmount(aux.WonBid); ok {
		r.WonBid = amount
	}
	return nil
}

func parseWonBidAmount(raw json.RawMessage) (float64, bool) {
	var bid struct {
		Amount json.RawMessage `json:"amount"`
	}
	if err := json.Unmarshal(raw, &bid); err != nil {
		return 0, false
	}
	return parseJSONFloat(bid.Amount)
}

func parseJSONFloat(raw json.RawMessage) (float64, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false
	}
	var numeric float64
	if err := json.Unmarshal(raw, &numeric); err == nil {
		return numeric, true
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		parsed, parseErr := strconv.ParseFloat(text, 64)
		if parseErr == nil {
			return parsed, true
		}
	}
	return 0, false
}

func (a *Auction) UnmarshalJSON(data []byte) error {
	type auctionAlias Auction
	var aux struct {
		auctionAlias
		CurrentPrice any `json:"current_price"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*a = Auction(aux.auctionAlias)
	switch v := aux.CurrentPrice.(type) {
	case float64:
		a.CurrentPrice = v
	case string:
		if v == "" {
			a.CurrentPrice = 0
			return nil
		}
		price, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		a.CurrentPrice = price
	case nil:
		a.CurrentPrice = 0
	}
	return nil
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

type fixedPricePurchaseResp struct {
	Code int `json:"code"`
	Data struct {
		OrderID int64 `json:"order_id"`
	} `json:"data"`
	OrderID int64 `json:"order_id"`
}

type idResp struct {
	ID   int64 `json:"id"`
	Data struct {
		ID int64 `json:"id"`
	} `json:"data"`
}

// ---------- 各步骤实现 ----------

func (c *Client) EnsureUsers(ctx context.Context, actors []Actor) StepResult {
	aggregate := StepResult{Step: "ensure_users", OK: true}
	start := time.Now()
	for _, actor := range actors {
		if actor.UserID <= 0 {
			continue
		}
		body := map[string]any{
			"id":     actor.UserID,
			"name":   actor.username(),
			"avatar": "",
		}
		step := c.doAs(ctx, "ensure_user", http.MethodPost, "/api/v1/users", Actor{}, body, nil, nil)
		if !step.OK {
			step.Step = "ensure_users"
			return step
		}
	}
	aggregate.DurationMs = time.Since(start).Milliseconds()
	return aggregate
}

// CreateProduct POST /api/v1/products
func (c *Client) CreateProduct(ctx context.Context, userID int64, req CreateProductReq) StepResult {
	return c.CreateProductAs(ctx, defaultActor(userID), req)
}

func (c *Client) CreateProductAs(ctx context.Context, actor Actor, req CreateProductReq) StepResult {
	var resp idResp
	path := "/api/v1/products"
	if actor.role() == RoleMerchant {
		path = "/api/v1/admin/products"
	}
	shouldPublish := req.Status == 1 && actor.role() == RoleMerchant
	if shouldPublish {
		req.Status = 0
	}
	if len(req.Images) == 0 {
		req.Images = []string{defaultFixtureProductImage}
	}
	if req.CategoryID == nil {
		req.CategoryID = fixtureCategoryID(DefaultFixtureProductCategoryID)
	}
	step := c.doAs(ctx, "create_product", http.MethodPost, path, actor, req, nil, &resp)
	step.RefID = firstNonZero(resp.ID, resp.Data.ID)
	if !step.OK || !shouldPublish || step.RefID == 0 {
		return step
	}

	publishPath := "/api/v1/products/" + strconv.FormatInt(step.RefID, 10) + "/publish"
	publishStep := c.doAs(ctx, "publish_product", http.MethodPost, publishPath, actor, nil, nil, nil)
	publishStep.RefID = step.RefID
	if !publishStep.OK {
		return publishStep
	}
	step.DurationMs += publishStep.DurationMs
	return step
}

func (c *Client) PublishProductAs(ctx context.Context, actor Actor, productID int64) StepResult {
	path := "/api/v1/products/" + strconv.FormatInt(productID, 10) + "/publish"
	step := c.doAs(ctx, "publish_product", http.MethodPost, path, actor, map[string]any{}, nil, nil)
	step.RefID = productID
	return step
}

// CreateAuction POST /api/v1/auctions
func (c *Client) CreateAuction(ctx context.Context, userID int64, req CreateAuctionReq) StepResult {
	return c.CreateAuctionAs(ctx, defaultActor(userID), req)
}

func (c *Client) CreateAuctionAs(ctx context.Context, actor Actor, req CreateAuctionReq) StepResult {
	var resp struct {
		ID int64 `json:"id"`
	}
	step := c.doAs(ctx, "create_auction", http.MethodPost, "/api/v1/auctions", actor, req, nil, &resp)
	step.RefID = resp.ID
	return step
}

func (c *Client) CreateAuctionRule(ctx context.Context, actor Actor, productID int64, req CreateAuctionRuleReq) StepResult {
	path := "/api/v1/products/" + strconv.FormatInt(productID, 10) + "/rules"
	step := c.doAs(ctx, "create_auction_rule", http.MethodPost, path, actor, req, nil, nil)
	step.RefID = productID
	return step
}

func (c *Client) CreateLiveStream(ctx context.Context, actor Actor, req CreateLiveStreamReq) StepResult {
	var resp idResp
	step := c.doAs(ctx, "create_live_stream", http.MethodPost, "/api/v1/admin/live-streams", actor, req, nil, &resp)
	step.RefID = firstNonZero(resp.ID, resp.Data.ID)
	return step
}

func (c *Client) StartLive(ctx context.Context, actor Actor, liveStreamID int64) StepResult {
	path := "/api/v1/live-streams/" + strconv.FormatInt(liveStreamID, 10) + "/start"
	step := c.doAs(ctx, "start_live", http.MethodPost, path, actor, nil, nil, nil)
	step.RefID = liveStreamID
	return step
}

func (c *Client) GetLiveStream(ctx context.Context, actor Actor, liveStreamID int64) (LiveStream, StepResult) {
	var resp struct {
		ID     int64 `json:"id"`
		Status any   `json:"status"`
		Data   struct {
			ID     int64 `json:"id"`
			Status any   `json:"status"`
		} `json:"data"`
	}
	path := "/api/v1/live-streams/" + strconv.FormatInt(liveStreamID, 10)
	step := c.doAs(ctx, "get_live_stream", http.MethodGet, path, actor, nil, nil, &resp)
	return LiveStream{
		ID:     firstNonZero(resp.ID, resp.Data.ID),
		Status: firstNonNil(resp.Status, resp.Data.Status),
	}, step
}

func (c *Client) CreateFixedPriceItem(ctx context.Context, actor Actor, req CreateFixedPriceItemReq) StepResult {
	var resp idResp
	step := c.doAs(ctx, "create_fixed_price_item", http.MethodPost, "/api/v1/fixed-price/items", actor, req, nil, &resp)
	step.RefID = firstNonZero(resp.ID, resp.Data.ID)
	return step
}

func (c *Client) ListFixedPriceItemsByLiveStream(ctx context.Context, actor Actor, liveStreamID int64) ([]FixedPriceItem, StepResult) {
	var resp struct {
		Items []FixedPriceItem `json:"items"`
		Data  struct {
			Items []FixedPriceItem `json:"items"`
		} `json:"data"`
	}
	path := "/api/v1/live-streams/" + strconv.FormatInt(liveStreamID, 10) + "/fixed-price/items"
	step := c.doAs(ctx, "list_fixed_price_items", http.MethodGet, path, actor, nil, nil, &resp)
	if len(resp.Items) > 0 {
		return resp.Items, step
	}
	return resp.Data.Items, step
}

func (c *Client) ListFixedPriceItemsByAuction(ctx context.Context, actor Actor, auctionID int64) ([]FixedPriceItem, StepResult) {
	var resp struct {
		Items []FixedPriceItem `json:"items"`
		Data  struct {
			Items []FixedPriceItem `json:"items"`
		} `json:"data"`
	}
	path := "/api/v1/auctions/" + strconv.FormatInt(auctionID, 10) + "/fixed-price/items"
	step := c.doAs(ctx, "list_fixed_price_items", http.MethodGet, path, actor, nil, nil, &resp)
	if len(resp.Items) > 0 {
		return resp.Items, step
	}
	return resp.Data.Items, step
}

// PlaceBid POST /api/v1/auctions/{id}/bids
func (c *Client) PlaceBid(ctx context.Context, userID, auctionID int64, amount float64) StepResult {
	body := map[string]any{"amount": amount, "user_id": userID}
	path := "/api/v1/auctions/" + strconv.FormatInt(auctionID, 10) + "/bids"
	return c.doAs(ctx, "bid", http.MethodPost, path, defaultActor(userID), body, nil, nil)
}

// GetAuction GET /api/v1/auctions/{id}
func (c *Client) GetAuction(ctx context.Context, auctionID int64) (Auction, StepResult) {
	var a Auction
	path := "/api/v1/auctions/" + strconv.FormatInt(auctionID, 10)
	step := c.doAs(ctx, "get_auction", http.MethodGet, path, Actor{}, nil, nil, &a)
	return a, step
}

func (c *Client) GetAuctionResult(ctx context.Context, auctionID int64) (AuctionResult, StepResult) {
	var resp struct {
		Code       int             `json:"code"`
		Message    string          `json:"message"`
		AuctionID  int64           `json:"auction_id"`
		ProductID  int64           `json:"product_id"`
		Status     int             `json:"status"`
		FinalPrice float64         `json:"final_price"`
		WinnerID   int64           `json:"winner_id"`
		WonBid     json.RawMessage `json:"won_bid"`
		Data       *AuctionResult  `json:"data"`
	}
	path := "/api/v1/auctions/" + strconv.FormatInt(auctionID, 10) + "/result"
	step := c.doAs(ctx, "get_auction_result", http.MethodGet, path, Actor{}, nil, nil, &resp)
	if step.OK && resp.Code != 0 && resp.Code != 200 {
		step.OK = false
		step.Message = firstNonEmpty(resp.Message, fmt.Sprintf("business code %d", resp.Code))
		return AuctionResult{}, step
	}
	if resp.Data != nil && resp.Data.AuctionID != 0 {
		return *resp.Data, step
	}
	result := AuctionResult{
		AuctionID:  resp.AuctionID,
		ProductID:  resp.ProductID,
		Status:     resp.Status,
		FinalPrice: resp.FinalPrice,
		WinnerID:   resp.WinnerID,
	}
	if amount, ok := parseJSONFloat(resp.WonBid); ok {
		result.WonBid = amount
	} else if amount, ok := parseWonBidAmount(resp.WonBid); ok {
		result.WonBid = amount
	}
	return result, step
}

// WaitAuctionStarted 轮询直到 status >= 1
func (c *Client) WaitAuctionStarted(ctx context.Context, auctionID int64, interval, timeout time.Duration) StepResult {
	return c.poll(ctx, "wait_started", auctionID, interval, timeout, func(a Auction) bool {
		return a.Status >= 1
	})
}

// WaitAuctionEnded 轮询直到 status >= 3；status=2 是延时中，不代表已生成订单。
func (c *Client) WaitAuctionEnded(ctx context.Context, auctionID int64, interval, timeout time.Duration) StepResult {
	return c.poll(ctx, "wait_ended", auctionID, interval, timeout, func(a Auction) bool {
		return a.Status >= 3
	})
}

// SubscribeSkyLamp POST /api/v1/sky-lamp/subscriptions
func (c *Client) SubscribeSkyLamp(ctx context.Context, userID, auctionID int64) StepResult {
	body := map[string]any{"auction_id": auctionID}
	var resp skyLampResp
	step := c.doAs(ctx, "skylamp_subscribe", http.MethodPost, "/api/v1/sky-lamp/subscriptions", defaultActor(userID), body, nil, &resp)
	step.RefID = resp.Subscription.ID
	return step
}

// FindOrdersByAuction 调 /api/v1/orders?user_id=winner，再按 auction_id 客户端过滤
func (c *Client) FindOrdersByAuction(ctx context.Context, winnerID, auctionID int64) ([]Order, StepResult) {
	var resp ordersResp
	path := "/api/v1/orders?user_id=" + strconv.FormatInt(winnerID, 10) + "&page=1&page_size=100"
	step := c.doAs(ctx, "find_orders", http.MethodGet, path, defaultActor(winnerID), nil, nil, &resp)
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

func (c *Client) FollowLiveStream(ctx context.Context, actor Actor, liveStreamID int64) StepResult {
	path := "/api/v1/live-streams/" + strconv.FormatInt(liveStreamID, 10) + "/follow"
	return c.doAs(ctx, "reminder", http.MethodPost, path, actor, nil, nil, nil)
}

func (c *Client) GetFollowStatus(ctx context.Context, actor Actor, liveStreamID int64) (bool, StepResult) {
	path := "/api/v1/live-streams/" + strconv.FormatInt(liveStreamID, 10) + "/follow-status"
	var resp struct {
		IsFollowing bool `json:"is_following"`
		Data        struct {
			IsFollowing bool `json:"is_following"`
		} `json:"data"`
	}
	step := c.doAs(ctx, "reminder", http.MethodGet, path, actor, nil, nil, &resp)
	if !step.OK {
		return false, step
	}
	if resp.Data.IsFollowing {
		return true, step
	}
	return resp.IsFollowing, step
}

func (c *Client) TopUpUserBalance(ctx context.Context, userID int64, amount string) (string, StepResult) {
	var resp struct {
		Balance string `json:"balance"`
		Data    struct {
			Balance string `json:"balance"`
		} `json:"data"`
	}
	step := c.doAs(ctx, "prepare", http.MethodPost, "/internal/test/user-balance", Actor{}, map[string]any{
		"user_id": userID,
		"amount":  amount,
	}, map[string]string{"X-Internal-Token": c.internalToken}, &resp)
	if !step.OK {
		return "", step
	}
	if resp.Data.Balance != "" {
		return resp.Data.Balance, step
	}
	return resp.Balance, step
}

func (c *Client) ShortenAuction(ctx context.Context, auctionID int64, remainingSeconds int) StepResult {
	step := c.doAs(ctx, "shorten_auction", http.MethodPost, "/internal/test/auctions/shorten", Actor{}, map[string]any{
		"auction_id":        auctionID,
		"remaining_seconds": remainingSeconds,
	}, map[string]string{"X-Internal-Token": c.internalToken}, nil)
	step.RefID = auctionID
	return step
}

func (c *Client) RestartLiveSession(ctx context.Context, liveStreamID int64) StepResult {
	path := "/internal/test/live-streams/" + strconv.FormatInt(liveStreamID, 10) + "/restart"
	step := c.doAs(ctx, "restart_live_session", http.MethodPost, path, Actor{}, map[string]any{}, map[string]string{
		"X-Internal-Token": c.internalToken,
	}, nil)
	step.RefID = liveStreamID
	return step
}

func (c *Client) CurrentAuctionByLiveStream(ctx context.Context, liveStreamID int64) (CurrentAuctionItem, StepResult) {
	var resp struct {
		Data struct {
			Items []CurrentAuctionItem `json:"items"`
		} `json:"data"`
	}
	step := c.doAs(ctx, "current_auction_by_live_stream", http.MethodPost, "/internal/auctions/current-by-live-streams", Actor{}, map[string]any{
		"live_stream_ids": []int64{liveStreamID},
	}, map[string]string{"X-Internal-Token": c.internalToken}, &resp)
	if !step.OK {
		return CurrentAuctionItem{}, step
	}
	for _, item := range resp.Data.Items {
		if item.LiveStreamID == liveStreamID {
			step.RefID = item.AuctionID
			return item, step
		}
	}
	return CurrentAuctionItem{}, step
}

func (c *Client) PurchaseFixedPriceItem(ctx context.Context, actor Actor, itemID int64, idemKey string) (int64, StepResult) {
	path := "/api/v1/fixed-price/items/" + strconv.FormatInt(itemID, 10) + "/purchase"
	var resp fixedPricePurchaseResp
	step := c.doAs(ctx, "fixed_price_purchase", http.MethodPost, path, actor, map[string]any{}, map[string]string{
		"X-Idempotency-Key": idemKey,
	}, &resp)
	if !step.OK {
		return 0, step
	}
	if resp.Data.OrderID != 0 {
		step.RefID = resp.Data.OrderID
		return resp.Data.OrderID, step
	}
	step.RefID = resp.OrderID
	return resp.OrderID, step
}

func (c *Client) GetMyFixedPricePurchase(ctx context.Context, actor Actor, itemID int64) (FixedPricePurchase, StepResult) {
	var resp struct {
		ID      int64 `json:"id"`
		ItemID  int64 `json:"item_id"`
		OrderID int64 `json:"order_id"`
		Data    struct {
			ID      int64 `json:"id"`
			ItemID  int64 `json:"item_id"`
			OrderID int64 `json:"order_id"`
		} `json:"data"`
	}
	path := "/api/v1/fixed-price/items/" + strconv.FormatInt(itemID, 10) + "/my-purchase"
	step := c.doAs(ctx, "get_my_fixed_price_purchase", http.MethodGet, path, actor, nil, nil, &resp)
	purchase := FixedPricePurchase{
		ID:      firstNonZero(resp.ID, resp.Data.ID),
		ItemID:  firstNonZero(resp.ItemID, resp.Data.ItemID),
		OrderID: firstNonZero(resp.OrderID, resp.Data.OrderID),
	}
	step.RefID = purchase.OrderID
	return purchase, step
}

func (c *Client) GetUserBalance(ctx context.Context, actor Actor) (string, StepResult) {
	var resp struct {
		AvailableAmount string `json:"available_amount"`
		Data            struct {
			AvailableAmount string `json:"available_amount"`
		} `json:"data"`
	}
	step := c.doAs(ctx, "get_user_balance", http.MethodGet, "/api/v1/user/balance", actor, nil, nil, &resp)
	return firstNonEmpty(resp.AvailableAmount, resp.Data.AvailableAmount), step
}

// ---------- 内部工具 ----------

// do 通用 HTTP 调用 + StepResult 装配
func (c *Client) do(ctx context.Context, step, method, path string, userID int64, body any, out any) StepResult {
	return c.doAs(ctx, step, method, path, defaultActor(userID), body, nil, out)
}

func (c *Client) doAs(ctx context.Context, step, method, path string, actor Actor, body any, extraHeaders map[string]string, out any) StepResult {
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
	if actor.UserID > 0 {
		req.Header.Set("X-User-ID", strconv.FormatInt(actor.UserID, 10))
		req.Header.Set("X-Username", actor.username())
		req.Header.Set("X-User-Role", actor.role())
		if c.jwtSecret != "" {
			token, err := actor.jwt(c.jwtSecret)
			if err != nil {
				res.Err = err
				res.Message = "jwt: " + err.Error()
				res.DurationMs = time.Since(start).Milliseconds()
				return res
			}
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
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
		res.Message = readHTTPErrorMessage(resp)
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

func readHTTPErrorMessage(resp *http.Response) string {
	fallback := fmt.Sprintf("HTTP %d", resp.StatusCode)
	if resp == nil || resp.Body == nil {
		return fallback
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil || len(data) == 0 {
		return fallback
	}
	var body struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return fallback
	}
	if body.Message != "" {
		return body.Message
	}
	if body.Error != "" {
		return body.Error
	}
	return fallback
}

func defaultActor(userID int64) Actor {
	if userID <= 0 {
		return Actor{}
	}
	return Actor{
		UserID:   userID,
		Username: "test_user_" + strconv.FormatInt(userID, 10),
		Role:     RoleUser,
	}
}

func (a Actor) username() string {
	if a.Username != "" {
		return a.Username
	}
	if a.UserID > 0 {
		return "test_user_" + strconv.FormatInt(a.UserID, 10)
	}
	return ""
}

func (a Actor) role() string {
	switch a.Role {
	case RoleMerchant:
		return RoleMerchant
	default:
		return RoleUser
	}
}

func (a Actor) roleCode() int {
	switch a.Role {
	case RoleMerchant:
		return 1
	default:
		return 0
	}
}

func (a Actor) jwt(secret string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  a.UserID,
		"username": a.username(),
		"role":     a.roleCode(),
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
		"nbf":      time.Now().Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func firstNonZero(values ...int64) int64 {
	for _, v := range values {
		if v != 0 {
			return v
		}
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func firstNonNil(values ...any) any {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
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
