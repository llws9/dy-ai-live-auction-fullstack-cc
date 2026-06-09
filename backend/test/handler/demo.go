package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/golang-jwt/jwt/v5"
	"github.com/shopspring/decimal"

	auctioncli "test-service/client/auction"
)

const (
	buyerAUserID   int64 = 9101
	buyerBUserID   int64 = 9102
	merchantUserID int64 = 9103
	adminUserID    int64 = 9104

	demoCategoryJewelryID int64 = 8
	demoCategoryArtID     int64 = 12
)

const (
	defaultConcurrentBidCount      = 6
	maxConcurrentBidCount          = 20
	defaultConcurrentBidIntervalMS = 80
	maxConcurrentBidIntervalMS     = 1000
)

type demoAuctionClient interface {
	GetAuction(ctx context.Context, auctionID int64) (auctioncli.Auction, auctioncli.StepResult)
	PlaceBid(ctx context.Context, userID, auctionID int64, amount float64) auctioncli.StepResult
	CreateProductAs(ctx context.Context, actor auctioncli.Actor, req auctioncli.CreateProductReq) auctioncli.StepResult
	PublishProductAs(ctx context.Context, actor auctioncli.Actor, productID int64) auctioncli.StepResult
	CreateAuctionRule(ctx context.Context, actor auctioncli.Actor, productID int64, req auctioncli.CreateAuctionRuleReq) auctioncli.StepResult
	CreateLiveStream(ctx context.Context, actor auctioncli.Actor, req auctioncli.CreateLiveStreamReq) auctioncli.StepResult
	StartLive(ctx context.Context, actor auctioncli.Actor, liveStreamID int64) auctioncli.StepResult
	FollowLiveStream(ctx context.Context, actor auctioncli.Actor, liveStreamID int64) auctioncli.StepResult
	CreateAuctionAs(ctx context.Context, actor auctioncli.Actor, req auctioncli.CreateAuctionReq) auctioncli.StepResult
	WaitAuctionStarted(ctx context.Context, auctionID int64, interval, timeout time.Duration) auctioncli.StepResult
	CreateFixedPriceItem(ctx context.Context, actor auctioncli.Actor, req auctioncli.CreateFixedPriceItemReq) auctioncli.StepResult
	SubscribeSkyLamp(ctx context.Context, userID, auctionID int64) auctioncli.StepResult
}

type demoInternalAuctionClient interface {
	TopUpUserBalance(ctx context.Context, userID int64, amount string) (string, auctioncli.StepResult)
	ShortenAuction(ctx context.Context, auctionID int64, remainingSeconds int) auctioncli.StepResult
	RestartLiveSession(ctx context.Context, liveStreamID int64) auctioncli.StepResult
	CurrentAuctionByLiveStream(ctx context.Context, liveStreamID int64) (auctioncli.CurrentAuctionItem, auctioncli.StepResult)
}

// DemoHandler 处理演示控制面板触发的同步业务动作。
type DemoHandler struct {
	bizCli      demoAuctionClient
	internalCli demoInternalAuctionClient
	jwtSecret   string
}

func NewDemoHandler(bizCli demoAuctionClient, internalCli demoInternalAuctionClient, jwtSecret string) *DemoHandler {
	return &DemoHandler{bizCli: bizCli, internalCli: internalCli, jwtSecret: jwtSecret}
}

type followBidRequest struct {
	AuctionID int64           `json:"auction_id"`
	Amount    json.RawMessage `json:"amount,omitempty"`
	Increment json.RawMessage `json:"increment,omitempty"`
}

type concurrentBidsRequest struct {
	AuctionID  int64           `json:"auction_id"`
	BidCount   int             `json:"bid_count,omitempty"`
	IntervalMS int             `json:"interval_ms,omitempty"`
	Increment  json.RawMessage `json:"increment,omitempty"`
}

type skyLampRequest struct {
	AuctionID int64 `json:"auction_id"`
}

type rechargeRequest struct {
	UserID int64  `json:"user_id"`
	Amount string `json:"amount"`
}

type merchantAuctionRequest struct {
	Mode string `json:"mode"`
}

type merchantFixedPriceRequest struct {
	AuctionID    int64 `json:"auction_id"`
	LiveStreamID int64 `json:"live_stream_id"`
}

type shortenAuctionRequest struct {
	AuctionID        int64 `json:"auction_id"`
	RemainingSeconds int   `json:"remaining_seconds"`
}

var demoMerchantSequence atomic.Int64

func zeroFollowBidAmount() decimal.Decimal {
	return decimal.Zero
}

func parseFollowBidAmount(raw string) (decimal.Decimal, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return zeroFollowBidAmount(), fmt.Errorf("empty amount")
	}
	amount, err := decimal.NewFromString(raw)
	if err != nil {
		return zeroFollowBidAmount(), fmt.Errorf("invalid amount %q", raw)
	}
	return amount, nil
}

func parseOptionalFollowBidAmount(raw json.RawMessage) (*decimal.Decimal, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return nil, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		text = string(raw)
	}
	amount, err := parseFollowBidAmount(text)
	if err != nil {
		return nil, err
	}
	return &amount, nil
}

func computeFollowBidAmount(current, start, increment decimal.Decimal, override *decimal.Decimal) decimal.Decimal {
	if override != nil {
		return *override
	}
	if increment.IsZero() {
		increment = decimal.NewFromInt(1)
	}
	baseline := current
	if start.GreaterThan(baseline) {
		baseline = start
	}
	return baseline.Add(increment)
}

func normalizeConcurrentBidsRequest(req *concurrentBidsRequest) error {
	if req.AuctionID <= 0 {
		return fmt.Errorf("auction_id is required")
	}
	if req.BidCount == 0 {
		req.BidCount = defaultConcurrentBidCount
	}
	if req.BidCount < 1 || req.BidCount > maxConcurrentBidCount {
		return fmt.Errorf("bid_count must be between 1 and %d", maxConcurrentBidCount)
	}
	if req.IntervalMS == 0 {
		req.IntervalMS = defaultConcurrentBidIntervalMS
	}
	if req.IntervalMS < 0 || req.IntervalMS > maxConcurrentBidIntervalMS {
		return fmt.Errorf("interval_ms must be between 0 and %d", maxConcurrentBidIntervalMS)
	}
	return nil
}

func effectiveConcurrentIncrement(requested *decimal.Decimal, ruleIncrement decimal.Decimal) decimal.Decimal {
	increment := ruleIncrement
	if requested != nil && requested.IsPositive() {
		increment = *requested
	}
	if ruleIncrement.IsPositive() && increment.LessThan(ruleIncrement) {
		increment = ruleIncrement
	}
	if !increment.IsPositive() {
		increment = decimal.NewFromInt(1)
	}
	return increment
}

func concurrentBidAmount(baseline, increment decimal.Decimal, index int) decimal.Decimal {
	return baseline.Add(increment.Mul(decimal.NewFromInt(int64(index + 1))))
}

func demoUserIDFromAuthorization(authHeader, jwtSecret string) (int64, error) {
	if strings.TrimSpace(jwtSecret) == "" {
		return 0, fmt.Errorf("jwt secret is not configured")
	}
	parts := strings.SplitN(strings.TrimSpace(authHeader), " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" || strings.TrimSpace(parts[1]) == "" {
		return 0, fmt.Errorf("authorization bearer token is required")
	}
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(strings.TrimSpace(parts[1]), claims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected jwt signing method")
		}
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return 0, fmt.Errorf("invalid jwt")
	}
	userID, err := int64Claim(claims["user_id"])
	if err != nil {
		return 0, fmt.Errorf("invalid user_id claim")
	}
	if !isAllowedDemoUserID(userID) {
		return 0, fmt.Errorf("user is not allowed to use demo endpoints")
	}
	return userID, nil
}

func int64Claim(value any) (int64, error) {
	switch v := value.(type) {
	case float64:
		i := int64(v)
		if float64(i) != v {
			return 0, fmt.Errorf("non-integer number")
		}
		return i, nil
	case json.Number:
		return v.Int64()
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("unsupported claim type")
	}
}

func isAllowedDemoUserID(userID int64) bool {
	switch userID {
	case buyerAUserID, buyerBUserID, merchantUserID, adminUserID:
		return true
	}

	for _, raw := range strings.FieldsFunc(os.Getenv("DEMO_ALLOWED_USER_IDS"), func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t'
	}) {
		allowedID, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		if err != nil {
			continue
		}
		if userID == allowedID {
			return true
		}
	}

	return false
}

func (h *DemoHandler) authorizeDemoRequest(c *app.RequestContext) bool {
	_, err := demoUserIDFromAuthorization(string(c.GetHeader("Authorization")), h.jwtSecret)
	if err != nil {
		c.JSON(401, map[string]any{"error": err.Error()})
		return false
	}
	return true
}

func validateRechargeRequest(userID int64, amount string) error {
	if userID != buyerAUserID && userID != buyerBUserID {
		return fmt.Errorf("recharge target must be demo buyer A or B")
	}
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return fmt.Errorf("amount is required")
	}
	parsed, err := decimal.NewFromString(amount)
	if err != nil {
		return fmt.Errorf("invalid amount")
	}
	if !parsed.IsPositive() {
		return fmt.Errorf("amount must be positive")
	}
	return nil
}

func validateMerchantAuctionMode(mode string) error {
	switch strings.TrimSpace(mode) {
	case "upcoming", "ongoing":
		return nil
	default:
		return fmt.Errorf("unsupported merchant auction mode")
	}
}

func validateDemoLiveStreamID(liveStreamID int64) error {
	if liveStreamID <= 0 {
		return fmt.Errorf("live_stream_id is required")
	}
	return nil
}

func validateDemoAuctionID(auctionID int64) error {
	if auctionID <= 0 {
		return fmt.Errorf("auction_id is required")
	}
	return nil
}

func validateShortenAuctionRequest(auctionID int64, remainingSeconds int) error {
	if auctionID <= 0 {
		return fmt.Errorf("auction_id is required")
	}
	if remainingSeconds <= 0 || remainingSeconds > 600 {
		return fmt.Errorf("remaining_seconds must be between 1 and 600")
	}
	return nil
}

func demoMerchantActor() auctioncli.Actor {
	return auctioncli.Actor{
		UserID: merchantUserID,
		Role:   auctioncli.RoleMerchant,
	}
}

func demoBuyerActors() []auctioncli.Actor {
	return []auctioncli.Actor{
		{UserID: buyerAUserID, Role: auctioncli.RoleUser},
		{UserID: buyerBUserID, Role: auctioncli.RoleUser},
	}
}

func nextDemoProductName(kind string) string {
	seq := demoMerchantSequence.Add(1)
	return fmt.Sprintf("DEMO_商家动作_%s_%d_%d", kind, time.Now().UnixNano(), seq)
}

func demoCategoryID(id int64) *int64 {
	return &id
}

func writeDemoStepError(c *app.RequestContext, step auctioncli.StepResult) {
	status := step.StatusCode
	if status < 400 {
		status = 400
	}
	c.JSON(status, map[string]any{"error": step.Message, "step": step.Step, "status": step.StatusCode})
}

func decimalToBidAmount(amount decimal.Decimal) (float64, error) {
	bidAmount, _ := amount.Float64()
	if math.IsInf(bidAmount, 0) || math.IsNaN(bidAmount) {
		return 0, fmt.Errorf("amount is outside supported bid range")
	}
	return bidAmount, nil
}

func isActiveLiveStreamAuctionConflict(step auctioncli.StepResult) bool {
	return step.StatusCode == 409 && strings.Contains(step.Message, "直播间")
}

func (h *DemoHandler) restartDemoLiveSession(ctx context.Context, c *app.RequestContext, liveStreamID int64) bool {
	if h.internalCli == nil {
		c.JSON(500, map[string]any{"error": "demo internal auction client is not configured"})
		return false
	}
	restarted := h.internalCli.RestartLiveSession(ctx, liveStreamID)
	if !restarted.OK {
		writeDemoStepError(c, restarted)
		return false
	}
	return true
}

func (h *DemoHandler) currentDemoAuction(ctx context.Context, c *app.RequestContext, liveStreamID int64) (auctioncli.CurrentAuctionItem, bool) {
	if h.internalCli == nil {
		c.JSON(500, map[string]any{"error": "demo internal auction client is not configured"})
		return auctioncli.CurrentAuctionItem{}, false
	}
	current, step := h.internalCli.CurrentAuctionByLiveStream(ctx, liveStreamID)
	if !step.OK {
		writeDemoStepError(c, step)
		return auctioncli.CurrentAuctionItem{}, false
	}
	return current, true
}

// PostFollowBid 以统一 seed 的买家B身份对指定拍卖发起一次跟价出价。
func (h *DemoHandler) PostFollowBid(ctx context.Context, c *app.RequestContext) {
	if !h.authorizeDemoRequest(c) {
		return
	}
	if h.bizCli == nil {
		c.JSON(500, map[string]any{"error": "demo auction client is not configured"})
		return
	}

	var req followBidRequest
	if err := json.Unmarshal(c.Request.Body(), &req); err != nil || req.AuctionID <= 0 {
		c.JSON(400, map[string]any{"error": "invalid auction_id"})
		return
	}

	override, err := parseOptionalFollowBidAmount(req.Amount)
	if err != nil {
		c.JSON(400, map[string]any{"error": "invalid amount"})
		return
	}
	increment := zeroFollowBidAmount()
	if len(req.Increment) > 0 {
		parsed, err := parseOptionalFollowBidAmount(req.Increment)
		if err != nil {
			c.JSON(400, map[string]any{"error": "invalid increment"})
			return
		}
		if parsed != nil {
			increment = *parsed
		}
	}

	current := zeroFollowBidAmount()
	start := zeroFollowBidAmount()
	if override == nil {
		auction, step := h.bizCli.GetAuction(ctx, req.AuctionID)
		if !step.OK {
			c.JSON(400, map[string]any{"error": step.Message, "status": step.StatusCode})
			return
		}
		current = decimal.NewFromFloat(auction.CurrentPrice)
		if auction.Rules != nil {
			start = auction.Rules.StartPrice
			if increment.IsZero() && auction.Rules.Increment.IsPositive() {
				increment = auction.Rules.Increment
			}
		}
	}

	amount := computeFollowBidAmount(current, start, increment, override)
	if !amount.IsPositive() {
		c.JSON(400, map[string]any{"error": "amount must be positive"})
		return
	}
	hlog.CtxInfof(ctx, "[demo] follow-bid auction=%d amount=%s as buyerB=%d", req.AuctionID, amount, buyerBUserID)
	// Existing auction SDK boundary accepts float64; keep business amount as decimal until this call.
	bidAmount, err := decimalToBidAmount(amount)
	if err != nil {
		c.JSON(400, map[string]any{"error": err.Error()})
		return
	}
	step := h.bizCli.PlaceBid(ctx, buyerBUserID, req.AuctionID, bidAmount)
	if !step.OK {
		c.JSON(400, map[string]any{"error": step.Message, "status": step.StatusCode})
		return
	}
	c.JSON(200, map[string]any{"ok": true, "auction_id": req.AuctionID, "buyer_user_id": buyerBUserID, "amount": amount.String()})
}

// PostConcurrentBids 以统一 seed 的买家B身份串行快速递增出价，稳定抬高当前拍卖价格。
func (h *DemoHandler) PostConcurrentBids(ctx context.Context, c *app.RequestContext) {
	if !h.authorizeDemoRequest(c) {
		return
	}
	if h.bizCli == nil {
		c.JSON(500, map[string]any{"error": "demo auction client is not configured"})
		return
	}

	var req concurrentBidsRequest
	if err := json.Unmarshal(c.Request.Body(), &req); err != nil {
		c.JSON(400, map[string]any{"error": "invalid concurrent bids request"})
		return
	}
	if err := normalizeConcurrentBidsRequest(&req); err != nil {
		c.JSON(400, map[string]any{"error": err.Error()})
		return
	}

	requestedIncrement, err := parseOptionalFollowBidAmount(req.Increment)
	if err != nil {
		c.JSON(400, map[string]any{"error": "invalid increment"})
		return
	}

	auction, step := h.bizCli.GetAuction(ctx, req.AuctionID)
	if !step.OK {
		c.JSON(400, map[string]any{"error": step.Message, "status": step.StatusCode})
		return
	}

	current := decimal.NewFromFloat(auction.CurrentPrice)
	start := decimal.Zero
	ruleIncrement := decimal.Zero
	capPrice := decimal.Zero
	if auction.Rules != nil {
		start = auction.Rules.StartPrice
		ruleIncrement = auction.Rules.Increment
		capPrice = auction.Rules.CapPrice
	}
	baseline := current
	if start.GreaterThan(baseline) {
		baseline = start
	}
	increment := effectiveConcurrentIncrement(requestedIncrement, ruleIncrement)

	successCount := 0
	failureCount := 0
	highestAmount := decimal.Zero
	lastError := ""

	for i := 0; i < req.BidCount; i++ {
		if i > 0 && req.IntervalMS > 0 {
			select {
			case <-ctx.Done():
				lastError = ctx.Err().Error()
				failureCount++
				i = req.BidCount
				continue
			case <-time.After(time.Duration(req.IntervalMS) * time.Millisecond):
			}
		}

		amount := concurrentBidAmount(baseline, increment, i)
		if capPrice.IsPositive() && amount.GreaterThanOrEqual(capPrice) {
			break
		}

		bidAmount, err := decimalToBidAmount(amount)
		if err != nil {
			lastError = err.Error()
			failureCount++
			continue
		}

		hlog.CtxInfof(ctx, "[demo] concurrent-bids auction=%d amount=%s as buyerB=%d", req.AuctionID, amount, buyerBUserID)
		step := h.bizCli.PlaceBid(ctx, buyerBUserID, req.AuctionID, bidAmount)
		if !step.OK {
			lastError = step.Message
			failureCount++
			continue
		}
		successCount++
		highestAmount = amount
	}

	status := 200
	ok := true
	if successCount == 0 {
		status = 400
		ok = false
		if lastError == "" {
			lastError = "no bid was placed"
		}
	}

	c.JSON(status, map[string]any{
		"ok":             ok,
		"auction_id":     req.AuctionID,
		"buyer_user_id":  buyerBUserID,
		"success_count":  successCount,
		"failure_count":  failureCount,
		"highest_amount": highestAmount.String(),
		"last_error":     lastError,
	})
}

// PostSkyLamp 以统一 seed 的买家B身份对指定拍卖开启点天灯。
func (h *DemoHandler) PostSkyLamp(ctx context.Context, c *app.RequestContext) {
	if !h.authorizeDemoRequest(c) {
		return
	}
	if h.bizCli == nil {
		c.JSON(500, map[string]any{"error": "demo auction client is not configured"})
		return
	}

	var req skyLampRequest
	if err := json.Unmarshal(c.Request.Body(), &req); err != nil || req.AuctionID <= 0 {
		c.JSON(400, map[string]any{"error": "invalid auction_id"})
		return
	}

	hlog.CtxInfof(ctx, "[demo] sky-lamp auction=%d as buyerB=%d", req.AuctionID, buyerBUserID)
	step := h.bizCli.SubscribeSkyLamp(ctx, buyerBUserID, req.AuctionID)
	if !step.OK {
		writeDemoStepError(c, step)
		return
	}
	c.JSON(200, map[string]any{"ok": true, "auction_id": req.AuctionID, "buyer_user_id": buyerBUserID, "subscription_id": step.RefID})
}

// PostRecharge 给指定用户充值演示余额，金额保持 decimal string 语义传递到 auction internal API。
func (h *DemoHandler) PostRecharge(ctx context.Context, c *app.RequestContext) {
	if !h.authorizeDemoRequest(c) {
		return
	}
	if h.internalCli == nil {
		c.JSON(500, map[string]any{"error": "demo internal auction client is not configured"})
		return
	}

	var req rechargeRequest
	if err := json.Unmarshal(c.Request.Body(), &req); err != nil {
		c.JSON(400, map[string]any{"error": "invalid recharge request"})
		return
	}
	req.Amount = strings.TrimSpace(req.Amount)
	if err := validateRechargeRequest(req.UserID, req.Amount); err != nil {
		c.JSON(400, map[string]any{"error": err.Error()})
		return
	}

	hlog.CtxInfof(ctx, "[demo] recharge user=%d amount=%s", req.UserID, req.Amount)
	balance, step := h.internalCli.TopUpUserBalance(ctx, req.UserID, req.Amount)
	if !step.OK {
		c.JSON(400, map[string]any{"error": step.Message, "status": step.StatusCode})
		return
	}
	c.JSON(200, map[string]any{"ok": true, "user_id": req.UserID, "amount": req.Amount, "balance": balance})
}

// PostShortenAuction 将当前演示竞拍剩余时间压缩到指定秒数，并由 auction-service 广播 time_sync。
func (h *DemoHandler) PostShortenAuction(ctx context.Context, c *app.RequestContext) {
	if !h.authorizeDemoRequest(c) {
		return
	}
	if h.internalCli == nil {
		c.JSON(500, map[string]any{"error": "demo internal auction client is not configured"})
		return
	}

	var req shortenAuctionRequest
	if err := json.Unmarshal(c.Request.Body(), &req); err != nil {
		c.JSON(400, map[string]any{"error": "invalid shorten auction request"})
		return
	}
	if err := validateShortenAuctionRequest(req.AuctionID, req.RemainingSeconds); err != nil {
		c.JSON(400, map[string]any{"error": err.Error()})
		return
	}

	hlog.CtxInfof(ctx, "[demo] shorten auction=%d remaining=%d", req.AuctionID, req.RemainingSeconds)
	step := h.internalCli.ShortenAuction(ctx, req.AuctionID, req.RemainingSeconds)
	if !step.OK {
		writeDemoStepError(c, step)
		return
	}
	c.JSON(200, map[string]any{
		"ok":                true,
		"auction_id":        req.AuctionID,
		"remaining_seconds": req.RemainingSeconds,
	})
}

// PostMerchantAuction 使用统一 seed 的商家账号创建一场 demo 竞拍。
func (h *DemoHandler) PostMerchantAuction(ctx context.Context, c *app.RequestContext) {
	if !h.authorizeDemoRequest(c) {
		return
	}
	if h.bizCli == nil {
		c.JSON(500, map[string]any{"error": "demo auction client is not configured"})
		return
	}
	var req merchantAuctionRequest
	if err := json.Unmarshal(c.Request.Body(), &req); err != nil {
		c.JSON(400, map[string]any{"error": "invalid merchant auction request"})
		return
	}
	req.Mode = strings.TrimSpace(req.Mode)
	if err := validateMerchantAuctionMode(req.Mode); err != nil {
		c.JSON(400, map[string]any{"error": err.Error()})
		return
	}

	actor := demoMerchantActor()
	live := h.bizCli.CreateLiveStream(ctx, actor, auctioncli.CreateLiveStreamReq{
		Name:        "Demo 商家直播间",
		Description: "Demo Console merchant auction fixture",
	})
	if !live.OK || live.RefID <= 0 {
		writeDemoStepError(c, live)
		return
	}
	for _, buyer := range demoBuyerActors() {
		follow := h.bizCli.FollowLiveStream(ctx, buyer, live.RefID)
		if !follow.OK {
			writeDemoStepError(c, follow)
			return
		}
	}
	if req.Mode == "ongoing" {
		current, ok := h.currentDemoAuction(ctx, c, live.RefID)
		if !ok {
			return
		}
		if current.AuctionID > 0 {
			if !h.restartDemoLiveSession(ctx, c, live.RefID) {
				return
			}
			c.JSON(200, map[string]any{
				"ok":             true,
				"mode":           req.Mode,
				"product_id":     current.ProductID,
				"live_stream_id": live.RefID,
				"auction_id":     current.AuctionID,
				"reused":         true,
				"message":        "当前直播间已有进行中的竞拍，已刷新开播提醒",
			})
			return
		}
	}

	product := h.bizCli.CreateProductAs(ctx, actor, auctioncli.CreateProductReq{
		Name:        nextDemoProductName(req.Mode),
		Description: "Demo Console merchant auction fixture",
		CategoryID:  demoCategoryID(demoCategoryArtID),
		Status:      0,
	})
	if !product.OK || product.RefID <= 0 {
		writeDemoStepError(c, product)
		return
	}

	const durationSec = 180
	rule := h.bizCli.CreateAuctionRule(ctx, actor, product.RefID, auctioncli.CreateAuctionRuleReq{
		StartPrice:         100,
		Increment:          10,
		Duration:           durationSec,
		DelayDuration:      5,
		MaxDelayTime:       30,
		TriggerDelayBefore: 10,
	})
	if !rule.OK {
		writeDemoStepError(c, rule)
		return
	}
	published := h.bizCli.PublishProductAs(ctx, actor, product.RefID)
	if !published.OK {
		writeDemoStepError(c, published)
		return
	}

	now := time.Now()
	startTime := now.Add(-time.Second)
	if req.Mode == "upcoming" {
		startTime = now.Add(time.Minute)
	}
	auction := h.bizCli.CreateAuctionAs(ctx, actor, auctioncli.CreateAuctionReq{
		ProductID:    product.RefID,
		LiveStreamID: live.RefID,
		StartPrice:   100,
		Increment:    10,
		Duration:     durationSec,
		StartTime:    &startTime,
	})
	if !auction.OK && req.Mode == "ongoing" && isActiveLiveStreamAuctionConflict(auction) {
		if !h.restartDemoLiveSession(ctx, c, live.RefID) {
			return
		}
		c.JSON(200, map[string]any{
			"ok":             true,
			"mode":           req.Mode,
			"product_id":     product.RefID,
			"live_stream_id": live.RefID,
			"auction_id":     0,
			"reused":         true,
			"message":        "当前直播间已有进行中的竞拍，已刷新开播提醒",
		})
		return
	}
	if !auction.OK || auction.RefID <= 0 {
		writeDemoStepError(c, auction)
		return
	}
	if req.Mode == "ongoing" {
		liveStarted := h.bizCli.StartLive(ctx, actor, live.RefID)
		if !liveStarted.OK {
			writeDemoStepError(c, liveStarted)
			return
		}
		if !h.restartDemoLiveSession(ctx, c, live.RefID) {
			return
		}
		started := h.bizCli.WaitAuctionStarted(ctx, auction.RefID, 100*time.Millisecond, 5*time.Second)
		if !started.OK {
			writeDemoStepError(c, started)
			return
		}
	}

	hlog.CtxInfof(ctx, "[demo] merchant auction mode=%s product=%d live_stream=%d auction=%d", req.Mode, product.RefID, live.RefID, auction.RefID)
	c.JSON(200, map[string]any{
		"ok":             true,
		"mode":           req.Mode,
		"product_id":     product.RefID,
		"live_stream_id": live.RefID,
		"auction_id":     auction.RefID,
		"start_time":     startTime.Format(time.RFC3339),
		"end_time":       startTime.Add(durationSec * time.Second).Format(time.RFC3339),
	})
}

// PostMerchantFixedPriceItem 为当前直播间创建一个 demo 一口价商品。
func (h *DemoHandler) PostMerchantFixedPriceItem(ctx context.Context, c *app.RequestContext) {
	if !h.authorizeDemoRequest(c) {
		return
	}
	if h.bizCli == nil {
		c.JSON(500, map[string]any{"error": "demo auction client is not configured"})
		return
	}
	var req merchantFixedPriceRequest
	if err := json.Unmarshal(c.Request.Body(), &req); err != nil {
		c.JSON(400, map[string]any{"error": "invalid merchant fixed-price request"})
		return
	}
	if err := validateDemoLiveStreamID(req.LiveStreamID); err != nil {
		c.JSON(400, map[string]any{"error": err.Error()})
		return
	}
	if err := validateDemoAuctionID(req.AuctionID); err != nil {
		c.JSON(400, map[string]any{"error": err.Error()})
		return
	}

	actor := demoMerchantActor()
	product := h.bizCli.CreateProductAs(ctx, actor, auctioncli.CreateProductReq{
		Name:        nextDemoProductName("fixed_price"),
		Description: "Demo Console fixed-price fixture",
		CategoryID:  demoCategoryID(demoCategoryJewelryID),
		Status:      0,
	})
	if !product.OK || product.RefID <= 0 {
		writeDemoStepError(c, product)
		return
	}
	published := h.bizCli.PublishProductAs(ctx, actor, product.RefID)
	if !published.OK {
		writeDemoStepError(c, published)
		return
	}

	const price = "99.00"
	const stock int64 = 10
	item := h.bizCli.CreateFixedPriceItem(ctx, actor, auctioncli.CreateFixedPriceItemReq{
		AuctionID:    req.AuctionID,
		LiveStreamID: req.LiveStreamID,
		ProductID:    product.RefID,
		Price:        price,
		Stock:        stock,
	})
	if !item.OK || item.RefID <= 0 {
		writeDemoStepError(c, item)
		return
	}

	hlog.CtxInfof(ctx, "[demo] merchant fixed-price live_stream=%d product=%d item=%d", req.LiveStreamID, product.RefID, item.RefID)
	c.JSON(200, map[string]any{
		"ok":             true,
		"product_id":     product.RefID,
		"live_stream_id": req.LiveStreamID,
		"item_id":        item.RefID,
		"price":          price,
		"stock":          stock,
	})
}
