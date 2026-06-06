package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

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
)

type demoAuctionClient interface {
	GetAuction(ctx context.Context, auctionID int64) (auctioncli.Auction, auctioncli.StepResult)
	PlaceBid(ctx context.Context, userID, auctionID int64, amount float64) auctioncli.StepResult
}

type demoInternalAuctionClient interface {
	TopUpUserBalance(ctx context.Context, userID int64, amount string) (string, auctioncli.StepResult)
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

type rechargeRequest struct {
	UserID int64  `json:"user_id"`
	Amount string `json:"amount"`
}

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

func computeFollowBidAmount(current, increment decimal.Decimal, override *decimal.Decimal) decimal.Decimal {
	if override != nil {
		return *override
	}
	if increment.IsZero() {
		increment = decimal.NewFromInt(1)
	}
	return current.Add(increment)
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
	default:
		return false
	}
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
	if userID != buyerBUserID {
		return fmt.Errorf("recharge target must be demo buyer B")
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

func decimalToBidAmount(amount decimal.Decimal) (float64, error) {
	bidAmount, _ := amount.Float64()
	if math.IsInf(bidAmount, 0) || math.IsNaN(bidAmount) {
		return 0, fmt.Errorf("amount is outside supported bid range")
	}
	return bidAmount, nil
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
	if override == nil {
		auction, step := h.bizCli.GetAuction(ctx, req.AuctionID)
		if !step.OK {
			c.JSON(400, map[string]any{"error": step.Message, "status": step.StatusCode})
			return
		}
		current = decimal.NewFromFloat(auction.CurrentPrice)
		if increment.IsZero() && auction.Rules != nil && auction.Rules.Increment.IsPositive() {
			increment = auction.Rules.Increment
		}
	}

	amount := computeFollowBidAmount(current, increment, override)
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
