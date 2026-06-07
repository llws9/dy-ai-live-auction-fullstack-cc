package handler

import (
	"context"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/golang-jwt/jwt/v5"
	"github.com/shopspring/decimal"

	auctioncli "test-service/client/auction"
)

func TestComputeFollowBidAmount(t *testing.T) {
	cases := []struct {
		name     string
		current  string
		incr     string
		override string
		want     string
	}{
		{"override wins", "100", "10", "500", "500"},
		{"current plus increment", "100", "10", "", "110"},
		{"zero current uses increment", "0", "5", "", "5"},
		{"empty increment defaults to 1", "100", "", "", "101"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cur := mustFollowBidAmount(t, c.current)
			incr := zeroFollowBidAmount()
			if c.incr != "" {
				incr = mustFollowBidAmount(t, c.incr)
			}
			var override *decimal.Decimal
			if c.override != "" {
				v := mustFollowBidAmount(t, c.override)
				override = &v
			}
			got := computeFollowBidAmount(cur, incr, override)
			want := mustFollowBidAmount(t, c.want)
			if !got.Equal(want) {
				t.Fatalf("computeFollowBidAmount(%s,%s,%v)=%s want %s", c.current, c.incr, override, got, want)
			}
		})
	}
}

func TestValidateRechargeRequest(t *testing.T) {
	cases := []struct {
		name    string
		userID  int64
		amount  string
		wantErr bool
	}{
		{"valid buyer A", buyerAUserID, "100.00", false},
		{"valid buyer B", buyerBUserID, "100.00", false},
		{"reject merchant", merchantUserID, "100.00", true},
		{"reject admin", adminUserID, "100.00", true},
		{"zero user", 0, "100.00", true},
		{"empty amount", buyerBUserID, "", true},
		{"non-positive amount", buyerBUserID, "0", true},
		{"negative amount", buyerBUserID, "-5", true},
		{"bad amount", buyerBUserID, "abc", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateRechargeRequest(c.userID, c.amount)
			if (err != nil) != c.wantErr {
				t.Fatalf("validateRechargeRequest(%d,%q) err=%v wantErr=%v", c.userID, c.amount, err, c.wantErr)
			}
		})
	}
}

func TestDemoUserIDFromAuthorization(t *testing.T) {
	const secret = "demo-secret"
	cases := []struct {
		name    string
		header  string
		secret  string
		want    int64
		wantErr bool
	}{
		{"valid demo buyer", "Bearer " + signDemoToken(t, secret, buyerAUserID), secret, buyerAUserID, false},
		{"valid demo admin", "Bearer " + signDemoToken(t, secret, adminUserID), secret, adminUserID, false},
		{"missing bearer", signDemoToken(t, secret, buyerAUserID), secret, 0, true},
		{"bad secret", "Bearer " + signDemoToken(t, secret, buyerAUserID), "other-secret", 0, true},
		{"non demo user", "Bearer " + signDemoToken(t, secret, 42), secret, 0, true},
		{"empty configured secret", "Bearer " + signDemoToken(t, secret, buyerAUserID), "", 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := demoUserIDFromAuthorization(c.header, c.secret)
			if (err != nil) != c.wantErr {
				t.Fatalf("demoUserIDFromAuthorization() err=%v wantErr=%v", err, c.wantErr)
			}
			if got != c.want {
				t.Fatalf("demoUserIDFromAuthorization()=%d want %d", got, c.want)
			}
		})
	}
}

func TestDecimalToBidAmountRejectsUnsupportedRange(t *testing.T) {
	_, err := decimalToBidAmount(decimal.New(1, 400))
	if err == nil {
		t.Fatalf("decimalToBidAmount() expected range error")
	}
}

func TestMerchantDemoValidateAuctionMode(t *testing.T) {
	for _, mode := range []string{"upcoming", "ongoing"} {
		if err := validateMerchantAuctionMode(mode); err != nil {
			t.Fatalf("validateMerchantAuctionMode(%q): %v", mode, err)
		}
	}
	if err := validateMerchantAuctionMode("ended"); err == nil {
		t.Fatalf("validateMerchantAuctionMode should reject unsupported mode")
	}
}

func TestMerchantDemoAuctionCreatesFreshProductsForRepeatedOngoingClicks(t *testing.T) {
	const secret = "demo-secret"
	fake := &fakeDemoAuctionClient{}
	h := NewDemoHandler(fake, nil, secret)

	for i := 0; i < 2; i++ {
		c := newDemoRequestContext(t, secret, `{"mode":"ongoing"}`)
		h.PostMerchantAuction(context.Background(), c)
		if c.Response.StatusCode() != 200 {
			t.Fatalf("response status=%d body=%s", c.Response.StatusCode(), c.Response.Body())
		}
	}

	if len(fake.productReqs) != 2 || len(fake.auctionReqs) != 2 {
		t.Fatalf("expected two products and auctions, got products=%d auctions=%d", len(fake.productReqs), len(fake.auctionReqs))
	}
	if len(fake.publishedProductIDs) != 2 {
		t.Fatalf("expected two published products, got %d", len(fake.publishedProductIDs))
	}
	if fake.auctionReqs[0].ProductID == fake.auctionReqs[1].ProductID {
		t.Fatalf("repeated clicks must create different demo products, got product_id=%d", fake.auctionReqs[0].ProductID)
	}
	if fake.publishedProductIDs[0] != fake.auctionReqs[0].ProductID || fake.publishedProductIDs[1] != fake.auctionReqs[1].ProductID {
		t.Fatalf("auction demo products must be published before auction creation, published=%v auctionReqs=%v", fake.publishedProductIDs, fake.auctionReqs)
	}
	if fake.waitStartedCalls != 2 {
		t.Fatalf("ongoing mode should wait for auction started twice, got %d", fake.waitStartedCalls)
	}
}

func TestMerchantDemoAuctionRejectsUnsupportedMode(t *testing.T) {
	const secret = "demo-secret"
	h := NewDemoHandler(&fakeDemoAuctionClient{}, nil, secret)
	c := newDemoRequestContext(t, secret, `{"mode":"ended"}`)

	h.PostMerchantAuction(context.Background(), c)

	if c.Response.StatusCode() != 400 {
		t.Fatalf("status=%d want 400", c.Response.StatusCode())
	}
}

func TestMerchantDemoFixedPriceUsesRequestedLiveStream(t *testing.T) {
	const secret = "demo-secret"
	fake := &fakeDemoAuctionClient{}
	h := NewDemoHandler(fake, nil, secret)
	c := newDemoRequestContext(t, secret, `{"live_stream_id":880301}`)

	h.PostMerchantFixedPriceItem(context.Background(), c)

	if c.Response.StatusCode() != 200 {
		t.Fatalf("response status=%d body=%s", c.Response.StatusCode(), c.Response.Body())
	}
	if len(fake.fixedReqs) != 1 {
		t.Fatalf("expected one fixed price request, got %d", len(fake.fixedReqs))
	}
	if fake.fixedReqs[0].LiveStreamID != 880301 {
		t.Fatalf("live_stream_id=%d want 880301", fake.fixedReqs[0].LiveStreamID)
	}
	if fake.fixedReqs[0].ProductID == 0 {
		t.Fatalf("fixed price item should use newly created demo product")
	}
	if len(fake.publishedProductIDs) != 1 || fake.publishedProductIDs[0] != fake.fixedReqs[0].ProductID {
		t.Fatalf("fixed price demo product must be published before listing, published=%v fixedReq=%+v", fake.publishedProductIDs, fake.fixedReqs[0])
	}
}

func TestDemoShortenAuctionCallsInternalClientWithTenSeconds(t *testing.T) {
	const secret = "demo-secret"
	internal := &fakeDemoInternalAuctionClient{}
	h := NewDemoHandler(nil, internal, secret)
	c := newDemoRequestContext(t, secret, `{"auction_id":3001,"remaining_seconds":10}`)

	h.PostShortenAuction(context.Background(), c)

	if c.Response.StatusCode() != 200 {
		t.Fatalf("response status=%d body=%s", c.Response.StatusCode(), c.Response.Body())
	}
	if internal.shortenAuctionID != 3001 {
		t.Fatalf("auction_id=%d want 3001", internal.shortenAuctionID)
	}
	if internal.shortenRemainingSeconds != 10 {
		t.Fatalf("remaining_seconds=%d want 10", internal.shortenRemainingSeconds)
	}
}

func TestDemoShortenAuctionRejectsNonDemoUser(t *testing.T) {
	const secret = "demo-secret"
	internal := &fakeDemoInternalAuctionClient{}
	h := NewDemoHandler(nil, internal, secret)
	c := app.NewContext(0)
	c.Request.Header.Set("Authorization", "Bearer "+signDemoToken(t, secret, 42))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.SetBodyString(`{"auction_id":3001,"remaining_seconds":10}`)

	h.PostShortenAuction(context.Background(), c)

	if c.Response.StatusCode() != 401 {
		t.Fatalf("status=%d want 401", c.Response.StatusCode())
	}
	if internal.shortenAuctionID != 0 {
		t.Fatalf("shorten should not be called for non-demo user")
	}
}

func mustFollowBidAmount(t *testing.T, raw string) decimal.Decimal {
	t.Helper()
	amount, err := parseFollowBidAmount(raw)
	if err != nil {
		t.Fatalf("parseFollowBidAmount(%q): %v", raw, err)
	}
	return amount
}

func signDemoToken(t *testing.T, secret string, userID int64) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  userID,
		"username": "demo",
		"role":     0,
		"exp":      time.Now().Add(time.Hour).Unix(),
		"iat":      time.Now().Unix(),
		"nbf":      time.Now().Unix(),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign demo token: %v", err)
	}
	return token
}

func newDemoRequestContext(t *testing.T, secret string, body string) *app.RequestContext {
	t.Helper()
	c := app.NewContext(0)
	c.Request.Header.Set("Authorization", "Bearer "+signDemoToken(t, secret, buyerAUserID))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.SetBodyString(body)
	return c
}

type fakeDemoAuctionClient struct {
	nextProductID       int64
	nextLiveStreamID    int64
	nextAuctionID       int64
	nextFixedID         int64
	waitStartedCalls    int
	productReqs         []auctioncli.CreateProductReq
	publishedProductIDs []int64
	auctionReqs         []auctioncli.CreateAuctionReq
	fixedReqs           []auctioncli.CreateFixedPriceItemReq
}

func (f *fakeDemoAuctionClient) nextID(counter *int64, base int64) int64 {
	if *counter == 0 {
		*counter = base
	}
	id := *counter
	*counter = *counter + 1
	return id
}

func (f *fakeDemoAuctionClient) ok(step string, refID int64) auctioncli.StepResult {
	return auctioncli.StepResult{Step: step, OK: true, RefID: refID, StatusCode: 200}
}

func (f *fakeDemoAuctionClient) GetAuction(_ context.Context, auctionID int64) (auctioncli.Auction, auctioncli.StepResult) {
	return auctioncli.Auction{ID: auctionID, CurrentPrice: 100, Rules: &auctioncli.AuctionRules{Increment: decimal.NewFromInt(10)}}, f.ok("get_auction", auctionID)
}

func (f *fakeDemoAuctionClient) PlaceBid(_ context.Context, _ int64, auctionID int64, _ float64) auctioncli.StepResult {
	return f.ok("place_bid", auctionID)
}

func (f *fakeDemoAuctionClient) CreateProductAs(_ context.Context, _ auctioncli.Actor, req auctioncli.CreateProductReq) auctioncli.StepResult {
	f.productReqs = append(f.productReqs, req)
	return f.ok("create_product", f.nextID(&f.nextProductID, 1001))
}

func (f *fakeDemoAuctionClient) PublishProductAs(_ context.Context, _ auctioncli.Actor, productID int64) auctioncli.StepResult {
	f.publishedProductIDs = append(f.publishedProductIDs, productID)
	return f.ok("publish_product", productID)
}

func (f *fakeDemoAuctionClient) CreateAuctionRule(_ context.Context, _ auctioncli.Actor, productID int64, _ auctioncli.CreateAuctionRuleReq) auctioncli.StepResult {
	return f.ok("create_auction_rule", productID)
}

func (f *fakeDemoAuctionClient) CreateLiveStream(_ context.Context, _ auctioncli.Actor, _ auctioncli.CreateLiveStreamReq) auctioncli.StepResult {
	return f.ok("create_live_stream", f.nextID(&f.nextLiveStreamID, 2001))
}

func (f *fakeDemoAuctionClient) CreateAuctionAs(_ context.Context, _ auctioncli.Actor, req auctioncli.CreateAuctionReq) auctioncli.StepResult {
	f.auctionReqs = append(f.auctionReqs, req)
	return f.ok("create_auction", f.nextID(&f.nextAuctionID, 3001))
}

func (f *fakeDemoAuctionClient) WaitAuctionStarted(_ context.Context, auctionID int64, _, _ time.Duration) auctioncli.StepResult {
	f.waitStartedCalls++
	return f.ok("wait_auction_started", auctionID)
}

func (f *fakeDemoAuctionClient) CreateFixedPriceItem(_ context.Context, _ auctioncli.Actor, req auctioncli.CreateFixedPriceItemReq) auctioncli.StepResult {
	f.fixedReqs = append(f.fixedReqs, req)
	return f.ok("create_fixed_price_item", f.nextID(&f.nextFixedID, 4001))
}

type fakeDemoInternalAuctionClient struct {
	shortenAuctionID        int64
	shortenRemainingSeconds int
}

func (f *fakeDemoInternalAuctionClient) TopUpUserBalance(_ context.Context, userID int64, amount string) (string, auctioncli.StepResult) {
	return amount, auctioncli.StepResult{Step: "top_up", OK: true, RefID: userID, StatusCode: 200}
}

func (f *fakeDemoInternalAuctionClient) ShortenAuction(_ context.Context, auctionID int64, remainingSeconds int) auctioncli.StepResult {
	f.shortenAuctionID = auctionID
	f.shortenRemainingSeconds = remainingSeconds
	return auctioncli.StepResult{Step: "shorten_auction", OK: true, RefID: auctionID, StatusCode: 200}
}
