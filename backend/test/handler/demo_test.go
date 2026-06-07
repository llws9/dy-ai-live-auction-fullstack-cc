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
		start    string
		incr     string
		override string
		want     string
	}{
		{"override wins", "100", "80", "10", "500", "500"},
		{"current plus increment", "100", "80", "10", "", "110"},
		{"zero current uses start plus increment", "0", "100", "10", "", "110"},
		{"current above start wins", "120", "100", "10", "", "130"},
		{"empty increment defaults to 1", "100", "80", "", "", "101"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cur := mustFollowBidAmount(t, c.current)
			start := mustFollowBidAmount(t, c.start)
			incr := zeroFollowBidAmount()
			if c.incr != "" {
				incr = mustFollowBidAmount(t, c.incr)
			}
			var override *decimal.Decimal
			if c.override != "" {
				v := mustFollowBidAmount(t, c.override)
				override = &v
			}
			got := computeFollowBidAmount(cur, start, incr, override)
			want := mustFollowBidAmount(t, c.want)
			if !got.Equal(want) {
				t.Fatalf("computeFollowBidAmount(%s,%s,%s,%v)=%s want %s", c.current, c.start, c.incr, override, got, want)
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

func TestMerchantDemoAuctionReusesMerchantLiveStreamForRepeatedOngoingClicks(t *testing.T) {
	const secret = "demo-secret"
	fake := &fakeDemoAuctionClient{}
	internal := &fakeDemoInternalAuctionClient{}
	h := NewDemoHandler(fake, internal, secret)

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
	for _, req := range fake.productReqs {
		if req.Status != 0 {
			t.Fatalf("merchant auction demo must create draft products before explicit publish, status=%d", req.Status)
		}
		if req.CategoryID == nil || *req.CategoryID != demoCategoryArtID {
			t.Fatalf("merchant auction demo category_id: want %d, got %v", demoCategoryArtID, req.CategoryID)
		}
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
	if len(fake.liveStreamReqs) != 2 {
		t.Fatalf("expected two get-or-create live stream requests, got %d", len(fake.liveStreamReqs))
	}
	for _, req := range fake.liveStreamReqs {
		if req.Name != "Demo 商家直播间" || req.ProductID != 0 {
			t.Fatalf("merchant auction demo must target a stable merchant live stream, req=%+v", req)
		}
	}
	if fake.auctionReqs[0].LiveStreamID != fake.auctionReqs[1].LiveStreamID {
		t.Fatalf("repeated clicks must reuse merchant live stream, got live_stream_ids=%d,%d", fake.auctionReqs[0].LiveStreamID, fake.auctionReqs[1].LiveStreamID)
	}
	if len(fake.startedLiveStreamIDs) != 2 {
		t.Fatalf("ongoing mode should start live streams twice, got %d", len(fake.startedLiveStreamIDs))
	}
	if fake.startedLiveStreamIDs[0] != fake.auctionReqs[0].LiveStreamID || fake.startedLiveStreamIDs[1] != fake.auctionReqs[1].LiveStreamID {
		t.Fatalf("ongoing demo must start the auction live streams, started=%v auctionReqs=%v", fake.startedLiveStreamIDs, fake.auctionReqs)
	}
	if len(internal.restartedLiveStreamIDs) != 2 {
		t.Fatalf("ongoing demo must restart live session for fresh reminder receipts, got %d", len(internal.restartedLiveStreamIDs))
	}
	if internal.restartedLiveStreamIDs[0] != fake.auctionReqs[0].LiveStreamID || internal.restartedLiveStreamIDs[1] != fake.auctionReqs[1].LiveStreamID {
		t.Fatalf("ongoing demo must restart the auction live streams, restarted=%v auctionReqs=%v", internal.restartedLiveStreamIDs, fake.auctionReqs)
	}
	wantFollowedUsers := []int64{buyerAUserID, buyerBUserID, buyerAUserID, buyerBUserID}
	if len(fake.followCalls) != len(wantFollowedUsers) {
		t.Fatalf("demo must auto-follow for buyers A/B on every click, got calls=%+v", fake.followCalls)
	}
	for i, wantUserID := range wantFollowedUsers {
		if fake.followCalls[i].userID != wantUserID {
			t.Fatalf("follow call %d user_id=%d want %d", i, fake.followCalls[i].userID, wantUserID)
		}
		if fake.followCalls[i].liveStreamID != fake.auctionReqs[i/2].LiveStreamID {
			t.Fatalf("follow call %d live_stream_id=%d want %d", i, fake.followCalls[i].liveStreamID, fake.auctionReqs[i/2].LiveStreamID)
		}
	}
	if len(fake.ruleReqs) != 2 {
		t.Fatalf("expected two auction rule requests, got %d", len(fake.ruleReqs))
	}
	for _, req := range fake.ruleReqs {
		if req.TriggerDelayBefore != 10 {
			t.Fatalf("demo anti-snipe window=%d want 10", req.TriggerDelayBefore)
		}
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

func TestMerchantDemoOngoingRestartsLiveSessionWhenActiveAuctionExists(t *testing.T) {
	const secret = "demo-secret"
	fake := &fakeDemoAuctionClient{createAuctionConflict: true}
	internal := &fakeDemoInternalAuctionClient{}
	h := NewDemoHandler(fake, internal, secret)
	c := newDemoRequestContext(t, secret, `{"mode":"ongoing"}`)

	h.PostMerchantAuction(context.Background(), c)

	if c.Response.StatusCode() != 200 {
		t.Fatalf("response status=%d body=%s", c.Response.StatusCode(), c.Response.Body())
	}
	if len(internal.restartedLiveStreamIDs) != 1 || internal.restartedLiveStreamIDs[0] != 2001 {
		t.Fatalf("expected demo to refresh live reminder session, restarted=%v", internal.restartedLiveStreamIDs)
	}
	if len(fake.startedLiveStreamIDs) != 0 {
		t.Fatalf("public start live should not be called after active auction conflict, started=%v", fake.startedLiveStreamIDs)
	}
	if fake.waitStartedCalls != 0 {
		t.Fatalf("wait started should not be called after active auction conflict")
	}
	if len(fake.followCalls) != 2 {
		t.Fatalf("demo must still ensure buyer follows before refreshing reminders, calls=%+v", fake.followCalls)
	}
}

func TestMerchantDemoFixedPriceUsesRequestedLiveStream(t *testing.T) {
	const secret = "demo-secret"
	fake := &fakeDemoAuctionClient{}
	h := NewDemoHandler(fake, nil, secret)
	c := newDemoRequestContext(t, secret, `{"auction_id":770501,"live_stream_id":880301}`)

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
	if fake.fixedReqs[0].AuctionID != 770501 {
		t.Fatalf("auction_id=%d want 770501", fake.fixedReqs[0].AuctionID)
	}
	if fake.fixedReqs[0].ProductID == 0 {
		t.Fatalf("fixed price item should use newly created demo product")
	}
	if len(fake.publishedProductIDs) != 1 || fake.publishedProductIDs[0] != fake.fixedReqs[0].ProductID {
		t.Fatalf("fixed price demo product must be published before listing, published=%v fixedReq=%+v", fake.publishedProductIDs, fake.fixedReqs[0])
	}
	if len(fake.productReqs) != 1 || fake.productReqs[0].Status != 0 {
		t.Fatalf("fixed price demo must create a draft product before explicit publish, productReqs=%+v", fake.productReqs)
	}
	if fake.productReqs[0].CategoryID == nil || *fake.productReqs[0].CategoryID != demoCategoryJewelryID {
		t.Fatalf("fixed price demo category_id: want %d, got %v", demoCategoryJewelryID, fake.productReqs[0].CategoryID)
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

func TestDemoSkyLampSubscribesBuyerB(t *testing.T) {
	const secret = "demo-secret"
	fake := &fakeDemoAuctionClient{}
	h := NewDemoHandler(fake, nil, secret)
	c := newDemoRequestContext(t, secret, `{"auction_id":3001}`)

	h.PostSkyLamp(context.Background(), c)

	if c.Response.StatusCode() != 200 {
		t.Fatalf("response status=%d body=%s", c.Response.StatusCode(), c.Response.Body())
	}
	if fake.skyLampUserID != buyerBUserID {
		t.Fatalf("sky lamp user_id=%d want %d", fake.skyLampUserID, buyerBUserID)
	}
	if fake.skyLampAuctionID != 3001 {
		t.Fatalf("sky lamp auction_id=%d want 3001", fake.skyLampAuctionID)
	}
}

func TestDemoSkyLampRejectsNonDemoUser(t *testing.T) {
	const secret = "demo-secret"
	fake := &fakeDemoAuctionClient{}
	h := NewDemoHandler(fake, nil, secret)
	c := app.NewContext(0)
	c.Request.Header.Set("Authorization", "Bearer "+signDemoToken(t, secret, 42))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.SetBodyString(`{"auction_id":3001}`)

	h.PostSkyLamp(context.Background(), c)

	if c.Response.StatusCode() != 401 {
		t.Fatalf("status=%d want 401", c.Response.StatusCode())
	}
	if fake.skyLampAuctionID != 0 {
		t.Fatalf("sky lamp should not be called for non-demo user")
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
	nextProductID        int64
	nextLiveStreamID     int64
	nextAuctionID        int64
	nextFixedID          int64
	waitStartedCalls     int
	startedLiveStreamIDs []int64
	productReqs          []auctioncli.CreateProductReq
	publishedProductIDs  []int64
	ruleReqs             []auctioncli.CreateAuctionRuleReq
	liveStreamReqs       []auctioncli.CreateLiveStreamReq
	auctionReqs          []auctioncli.CreateAuctionReq
	fixedReqs            []auctioncli.CreateFixedPriceItemReq
	skyLampUserID        int64
	skyLampAuctionID     int64
	followCalls          []struct {
		userID       int64
		liveStreamID int64
	}
	createAuctionConflict bool
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

func (f *fakeDemoAuctionClient) CreateAuctionRule(_ context.Context, _ auctioncli.Actor, productID int64, req auctioncli.CreateAuctionRuleReq) auctioncli.StepResult {
	f.ruleReqs = append(f.ruleReqs, req)
	return f.ok("create_auction_rule", productID)
}

func (f *fakeDemoAuctionClient) CreateLiveStream(_ context.Context, _ auctioncli.Actor, req auctioncli.CreateLiveStreamReq) auctioncli.StepResult {
	f.liveStreamReqs = append(f.liveStreamReqs, req)
	if f.nextLiveStreamID == 0 {
		f.nextLiveStreamID = 2001
	}
	return f.ok("create_live_stream", f.nextLiveStreamID)
}

func (f *fakeDemoAuctionClient) CreateAuctionAs(_ context.Context, _ auctioncli.Actor, req auctioncli.CreateAuctionReq) auctioncli.StepResult {
	f.auctionReqs = append(f.auctionReqs, req)
	if f.createAuctionConflict {
		return auctioncli.StepResult{Step: "create_auction", OK: false, StatusCode: 409, Message: "当前直播间已有待开始或进行中的竞拍场次"}
	}
	return f.ok("create_auction", f.nextID(&f.nextAuctionID, 3001))
}

func (f *fakeDemoAuctionClient) StartLive(_ context.Context, _ auctioncli.Actor, liveStreamID int64) auctioncli.StepResult {
	f.startedLiveStreamIDs = append(f.startedLiveStreamIDs, liveStreamID)
	return f.ok("start_live", liveStreamID)
}

func (f *fakeDemoAuctionClient) FollowLiveStream(_ context.Context, actor auctioncli.Actor, liveStreamID int64) auctioncli.StepResult {
	f.followCalls = append(f.followCalls, struct {
		userID       int64
		liveStreamID int64
	}{userID: actor.UserID, liveStreamID: liveStreamID})
	return f.ok("follow_live_stream", liveStreamID)
}

func (f *fakeDemoAuctionClient) WaitAuctionStarted(_ context.Context, auctionID int64, _, _ time.Duration) auctioncli.StepResult {
	f.waitStartedCalls++
	return f.ok("wait_auction_started", auctionID)
}

func (f *fakeDemoAuctionClient) CreateFixedPriceItem(_ context.Context, _ auctioncli.Actor, req auctioncli.CreateFixedPriceItemReq) auctioncli.StepResult {
	f.fixedReqs = append(f.fixedReqs, req)
	return f.ok("create_fixed_price_item", f.nextID(&f.nextFixedID, 4001))
}

func (f *fakeDemoAuctionClient) SubscribeSkyLamp(_ context.Context, userID, auctionID int64) auctioncli.StepResult {
	f.skyLampUserID = userID
	f.skyLampAuctionID = auctionID
	return f.ok("skylamp_subscribe", 5001)
}

type fakeDemoInternalAuctionClient struct {
	shortenAuctionID        int64
	shortenRemainingSeconds int
	restartedLiveStreamIDs  []int64
}

func (f *fakeDemoInternalAuctionClient) TopUpUserBalance(_ context.Context, userID int64, amount string) (string, auctioncli.StepResult) {
	return amount, auctioncli.StepResult{Step: "top_up", OK: true, RefID: userID, StatusCode: 200}
}

func (f *fakeDemoInternalAuctionClient) ShortenAuction(_ context.Context, auctionID int64, remainingSeconds int) auctioncli.StepResult {
	f.shortenAuctionID = auctionID
	f.shortenRemainingSeconds = remainingSeconds
	return auctioncli.StepResult{Step: "shorten_auction", OK: true, RefID: auctionID, StatusCode: 200}
}

func (f *fakeDemoInternalAuctionClient) RestartLiveSession(_ context.Context, liveStreamID int64) auctioncli.StepResult {
	f.restartedLiveStreamIDs = append(f.restartedLiveStreamIDs, liveStreamID)
	return auctioncli.StepResult{Step: "restart_live_session", OK: true, RefID: liveStreamID, StatusCode: 200}
}
