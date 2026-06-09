package handler

import (
	"context"
	"encoding/json"
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

func TestDemoUserIDFromAuthorizationAllowsConfiguredLegacyDemoUsers(t *testing.T) {
	const secret = "demo-secret"
	t.Setenv("DEMO_ALLOWED_USER_IDS", "4,5,6")

	got, err := demoUserIDFromAuthorization("Bearer "+signDemoToken(t, secret, 6), secret)
	if err != nil {
		t.Fatalf("demoUserIDFromAuthorization() err=%v", err)
	}
	if got != 6 {
		t.Fatalf("demoUserIDFromAuthorization()=%d want 6", got)
	}
}

func TestDecimalToBidAmountRejectsUnsupportedRange(t *testing.T) {
	_, err := decimalToBidAmount(decimal.New(1, 400))
	if err == nil {
		t.Fatalf("decimalToBidAmount() expected range error")
	}
}

func TestPostConcurrentBidsRejectsMissingAuctionID(t *testing.T) {
	const secret = "demo-secret"
	h := NewDemoHandler(&fakeDemoAuctionClient{}, nil, secret)
	c := newDemoRequestContext(t, secret, `{"bid_count":3}`)

	h.PostConcurrentBids(context.Background(), c)

	if c.Response.StatusCode() != 400 {
		t.Fatalf("status=%d want 400 body=%s", c.Response.StatusCode(), c.Response.Body())
	}
}

func TestPostConcurrentBidsRejectsBidCountAboveLimit(t *testing.T) {
	const secret = "demo-secret"
	fake := &fakeDemoAuctionClient{}
	h := NewDemoHandler(fake, nil, secret)
	c := newDemoRequestContext(t, secret, `{"auction_id":77,"bid_count":21,"interval_ms":0}`)

	h.PostConcurrentBids(context.Background(), c)

	if c.Response.StatusCode() != 400 {
		t.Fatalf("status=%d want 400 body=%s", c.Response.StatusCode(), c.Response.Body())
	}
	if len(fake.bidCalls) != 0 {
		t.Fatalf("invalid bid_count must not place bids, got calls=%+v", fake.bidCalls)
	}
}

func TestNormalizeConcurrentBidsRequestDefaultsMissingIntervalAndPreservesExplicitZero(t *testing.T) {
	var missing concurrentBidsRequest
	if err := json.Unmarshal([]byte(`{"auction_id":77}`), &missing); err != nil {
		t.Fatalf("unmarshal missing interval: %v", err)
	}
	if err := normalizeConcurrentBidsRequest(&missing); err != nil {
		t.Fatalf("normalize missing interval: %v", err)
	}
	if got := concurrentBidIntervalMS(missing); got != defaultConcurrentBidIntervalMS {
		t.Fatalf("missing interval_ms normalized to %d, want %d", got, defaultConcurrentBidIntervalMS)
	}

	var explicitZero concurrentBidsRequest
	if err := json.Unmarshal([]byte(`{"auction_id":77,"interval_ms":0}`), &explicitZero); err != nil {
		t.Fatalf("unmarshal explicit zero interval: %v", err)
	}
	if err := normalizeConcurrentBidsRequest(&explicitZero); err != nil {
		t.Fatalf("normalize explicit zero interval: %v", err)
	}
	if got := concurrentBidIntervalMS(explicitZero); got != 0 {
		t.Fatalf("explicit interval_ms=0 normalized to %d, want 0", got)
	}
}

func TestPostConcurrentBidsPlacesSerialIncrementalBids(t *testing.T) {
	const secret = "demo-secret"
	fake := &fakeDemoAuctionClient{
		auction: auctioncli.Auction{
			ID:           77,
			CurrentPrice: 100,
			Rules: &auctioncli.AuctionRules{
				StartPrice: decimal.NewFromInt(80),
				Increment:  decimal.NewFromInt(10),
			},
		},
	}
	h := NewDemoHandler(fake, nil, secret)
	c := newDemoRequestContext(t, secret, `{"auction_id":77,"bid_count":3,"interval_ms":0}`)

	h.PostConcurrentBids(context.Background(), c)

	if c.Response.StatusCode() != 200 {
		t.Fatalf("status=%d want 200 body=%s", c.Response.StatusCode(), c.Response.Body())
	}
	wantAmounts := []float64{110, 120, 130}
	if len(fake.bidCalls) != len(wantAmounts) {
		t.Fatalf("bid calls=%d want %d", len(fake.bidCalls), len(wantAmounts))
	}
	for i, want := range wantAmounts {
		if fake.bidCalls[i].userID != buyerBUserID {
			t.Fatalf("call %d user_id=%d want %d", i, fake.bidCalls[i].userID, buyerBUserID)
		}
		if fake.bidCalls[i].auctionID != 77 {
			t.Fatalf("call %d auction_id=%d want 77", i, fake.bidCalls[i].auctionID)
		}
		if fake.bidCalls[i].amount != want {
			t.Fatalf("call %d amount=%f want %f", i, fake.bidCalls[i].amount, want)
		}
	}
}

func TestPostConcurrentBidsExplicitZeroIntervalDoesNotWait(t *testing.T) {
	const secret = "demo-secret"
	fake := &fakeDemoAuctionClient{
		auction: auctioncli.Auction{
			ID:           77,
			CurrentPrice: 100,
			Rules:        &auctioncli.AuctionRules{Increment: decimal.NewFromInt(10)},
		},
	}
	h := NewDemoHandler(fake, nil, secret)
	c := newDemoRequestContext(t, secret, `{"auction_id":77,"bid_count":3,"interval_ms":0}`)

	startedAt := time.Now()
	h.PostConcurrentBids(context.Background(), c)
	elapsed := time.Since(startedAt)

	if c.Response.StatusCode() != 200 {
		t.Fatalf("status=%d want 200 body=%s", c.Response.StatusCode(), c.Response.Body())
	}
	if elapsed >= time.Duration(defaultConcurrentBidIntervalMS)*time.Millisecond {
		t.Fatalf("explicit interval_ms=0 should not wait, elapsed=%s", elapsed)
	}
}

func TestPostConcurrentBidsStopsBeforeCapPrice(t *testing.T) {
	const secret = "demo-secret"
	fake := &fakeDemoAuctionClient{
		auction: auctioncli.Auction{
			ID:           77,
			CurrentPrice: 100,
			Rules: &auctioncli.AuctionRules{
				StartPrice: decimal.NewFromInt(80),
				Increment:  decimal.NewFromInt(10),
				CapPrice:   decimal.NewFromInt(125),
			},
		},
	}
	h := NewDemoHandler(fake, nil, secret)
	c := newDemoRequestContext(t, secret, `{"auction_id":77,"bid_count":5,"interval_ms":0}`)

	h.PostConcurrentBids(context.Background(), c)

	if c.Response.StatusCode() != 200 {
		t.Fatalf("status=%d want 200 body=%s", c.Response.StatusCode(), c.Response.Body())
	}
	wantAmounts := []float64{110, 120}
	if len(fake.bidCalls) != len(wantAmounts) {
		t.Fatalf("bid calls=%d want %d", len(fake.bidCalls), len(wantAmounts))
	}
	for i, want := range wantAmounts {
		if fake.bidCalls[i].amount != want {
			t.Fatalf("call %d amount=%f want %f", i, fake.bidCalls[i].amount, want)
		}
	}
}

func TestPostConcurrentBidsPropagatesGetAuctionFailureStatus(t *testing.T) {
	const secret = "demo-secret"
	fake := &fakeDemoAuctionClient{
		getStep: auctioncli.StepResult{Step: "get_auction", OK: false, StatusCode: 500, Message: "auction upstream unavailable"},
	}
	h := NewDemoHandler(fake, nil, secret)
	c := newDemoRequestContext(t, secret, `{"auction_id":77,"bid_count":2,"interval_ms":0}`)

	h.PostConcurrentBids(context.Background(), c)

	if c.Response.StatusCode() != 500 {
		t.Fatalf("status=%d want 500 body=%s", c.Response.StatusCode(), c.Response.Body())
	}
	var body map[string]any
	if err := json.Unmarshal(c.Response.Body(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["step"] != "get_auction" || body["error"] != "auction upstream unavailable" {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestPostConcurrentBidsReturnsOKWhenPartiallyFailed(t *testing.T) {
	const secret = "demo-secret"
	fake := &fakeDemoAuctionClient{
		auction: auctioncli.Auction{
			ID:           77,
			CurrentPrice: 100,
			Rules:        &auctioncli.AuctionRules{Increment: decimal.NewFromInt(10)},
		},
		bidResults: []auctioncli.StepResult{
			{Step: "bid", OK: false, StatusCode: 400, Message: "出价过于频繁，请稍后再试"},
			{Step: "bid", OK: true, StatusCode: 200},
		},
	}
	h := NewDemoHandler(fake, nil, secret)
	c := newDemoRequestContext(t, secret, `{"auction_id":77,"bid_count":2,"interval_ms":0}`)

	h.PostConcurrentBids(context.Background(), c)

	if c.Response.StatusCode() != 200 {
		t.Fatalf("status=%d want 200 body=%s", c.Response.StatusCode(), c.Response.Body())
	}
	body := decodeConcurrentBidsResponse(t, c.Response.Body())
	if !body.OK || body.SuccessCount != 1 || body.FailureCount != 1 {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestPostConcurrentBidsReturnsBadRequestWhenAllFailed(t *testing.T) {
	const secret = "demo-secret"
	fake := &fakeDemoAuctionClient{
		auction: auctioncli.Auction{
			ID:           77,
			CurrentPrice: 100,
			Rules:        &auctioncli.AuctionRules{Increment: decimal.NewFromInt(10)},
		},
		bidResults: []auctioncli.StepResult{
			{Step: "bid", OK: false, StatusCode: 400, Message: "竞拍已结束，无法出价"},
			{Step: "bid", OK: false, StatusCode: 400, Message: "竞拍已结束，无法出价"},
		},
	}
	h := NewDemoHandler(fake, nil, secret)
	c := newDemoRequestContext(t, secret, `{"auction_id":77,"bid_count":2,"interval_ms":0}`)

	h.PostConcurrentBids(context.Background(), c)

	if c.Response.StatusCode() != 400 {
		t.Fatalf("status=%d want 400 body=%s", c.Response.StatusCode(), c.Response.Body())
	}
	body := decodeConcurrentBidsResponse(t, c.Response.Body())
	if body.OK || body.SuccessCount != 0 || body.LastError != "竞拍已结束，无法出价" {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestPostConcurrentBidsReturnsLastFailureStatusWhenAllFailed(t *testing.T) {
	const secret = "demo-secret"
	fake := &fakeDemoAuctionClient{
		auction: auctioncli.Auction{
			ID:           77,
			CurrentPrice: 100,
			Rules:        &auctioncli.AuctionRules{Increment: decimal.NewFromInt(10)},
		},
		bidResults: []auctioncli.StepResult{
			{Step: "bid", OK: false, StatusCode: 500, Message: "auction upstream unavailable"},
			{Step: "bid", OK: false, StatusCode: 409, Message: "竞拍已结束，无法出价"},
		},
	}
	h := NewDemoHandler(fake, nil, secret)
	c := newDemoRequestContext(t, secret, `{"auction_id":77,"bid_count":2,"interval_ms":0}`)

	h.PostConcurrentBids(context.Background(), c)

	if c.Response.StatusCode() != 409 {
		t.Fatalf("status=%d want 409 body=%s", c.Response.StatusCode(), c.Response.Body())
	}
	body := decodeConcurrentBidsResponse(t, c.Response.Body())
	if body.OK || body.SuccessCount != 0 || body.LastError != "竞拍已结束，无法出价" {
		t.Fatalf("unexpected response: %+v", body)
	}
}

type concurrentBidsResponseBody struct {
	OK           bool   `json:"ok"`
	SuccessCount int    `json:"success_count"`
	FailureCount int    `json:"failure_count"`
	LastError    string `json:"last_error"`
}

func decodeConcurrentBidsResponse(t *testing.T, raw []byte) concurrentBidsResponseBody {
	t.Helper()
	var body concurrentBidsResponseBody
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body
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
	internal := &fakeDemoInternalAuctionClient{
		currentAuctionAfterCalls: 2,
		currentAuction: auctioncli.CurrentAuctionItem{
			LiveStreamID: 2001,
			AuctionID:    3001,
			ProductID:    1001,
			Status:       1,
		},
	}
	h := NewDemoHandler(fake, internal, secret)

	for i := 0; i < 2; i++ {
		c := newDemoRequestContext(t, secret, `{"mode":"ongoing"}`)
		h.PostMerchantAuction(context.Background(), c)
		if c.Response.StatusCode() != 200 {
			t.Fatalf("response status=%d body=%s", c.Response.StatusCode(), c.Response.Body())
		}
	}

	if len(fake.productReqs) != 1 || len(fake.auctionReqs) != 1 {
		t.Fatalf("expected one product and auction, got products=%d auctions=%d", len(fake.productReqs), len(fake.auctionReqs))
	}
	if len(fake.publishedProductIDs) != 1 {
		t.Fatalf("expected one published product, got %d", len(fake.publishedProductIDs))
	}
	for _, req := range fake.productReqs {
		if req.Status != 0 {
			t.Fatalf("merchant auction demo must create draft products before explicit publish, status=%d", req.Status)
		}
		if req.CategoryID == nil || *req.CategoryID != demoCategoryArtID {
			t.Fatalf("merchant auction demo category_id: want %d, got %v", demoCategoryArtID, req.CategoryID)
		}
	}
	if fake.publishedProductIDs[0] != fake.auctionReqs[0].ProductID {
		t.Fatalf("auction demo products must be published before auction creation, published=%v auctionReqs=%v", fake.publishedProductIDs, fake.auctionReqs)
	}
	if fake.waitStartedCalls != 1 {
		t.Fatalf("ongoing mode should wait only for the newly-created auction, got %d", fake.waitStartedCalls)
	}
	if len(fake.liveStreamReqs) != 2 {
		t.Fatalf("expected two get-or-create live stream requests, got %d", len(fake.liveStreamReqs))
	}
	for _, req := range fake.liveStreamReqs {
		if req.Name != "Demo 商家直播间" || req.ProductID != 0 {
			t.Fatalf("merchant auction demo must target a stable merchant live stream, req=%+v", req)
		}
	}
	if len(fake.startedLiveStreamIDs) != 1 {
		t.Fatalf("ongoing mode should start only the newly-created auction live stream, got %d", len(fake.startedLiveStreamIDs))
	}
	if fake.startedLiveStreamIDs[0] != fake.auctionReqs[0].LiveStreamID {
		t.Fatalf("ongoing demo must start the auction live streams, started=%v auctionReqs=%v", fake.startedLiveStreamIDs, fake.auctionReqs)
	}
	if len(internal.restartedLiveStreamIDs) != 2 {
		t.Fatalf("ongoing demo must restart live session for fresh reminder receipts, got %d", len(internal.restartedLiveStreamIDs))
	}
	if internal.restartedLiveStreamIDs[0] != fake.auctionReqs[0].LiveStreamID || internal.restartedLiveStreamIDs[1] != fake.auctionReqs[0].LiveStreamID {
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
		if fake.followCalls[i].liveStreamID != fake.auctionReqs[0].LiveStreamID {
			t.Fatalf("follow call %d live_stream_id=%d want %d", i, fake.followCalls[i].liveStreamID, fake.auctionReqs[0].LiveStreamID)
		}
	}
	if len(fake.ruleReqs) != 1 {
		t.Fatalf("expected one auction rule request, got %d", len(fake.ruleReqs))
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
	auction              auctioncli.Auction
	getStep              auctioncli.StepResult
	bidResults           []auctioncli.StepResult
	bidCalls             []struct {
		userID    int64
		auctionID int64
		amount    float64
	}
	skyLampUserID    int64
	skyLampAuctionID int64
	followCalls      []struct {
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
	if f.getStep.Step != "" {
		return f.auction, f.getStep
	}
	if f.auction.ID != 0 {
		return f.auction, f.ok("get_auction", f.auction.ID)
	}
	return auctioncli.Auction{ID: auctionID, CurrentPrice: 100, Rules: &auctioncli.AuctionRules{Increment: decimal.NewFromInt(10)}}, f.ok("get_auction", auctionID)
}

func (f *fakeDemoAuctionClient) PlaceBid(_ context.Context, userID int64, auctionID int64, amount float64) auctioncli.StepResult {
	f.bidCalls = append(f.bidCalls, struct {
		userID    int64
		auctionID int64
		amount    float64
	}{userID: userID, auctionID: auctionID, amount: amount})
	if len(f.bidResults) > 0 {
		result := f.bidResults[0]
		f.bidResults = f.bidResults[1:]
		return result
	}
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
	shortenAuctionID         int64
	shortenRemainingSeconds  int
	restartedLiveStreamIDs   []int64
	currentAuctionCalls      int
	currentAuctionAfterCalls int
	currentAuction           auctioncli.CurrentAuctionItem
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

func (f *fakeDemoInternalAuctionClient) CurrentAuctionByLiveStream(_ context.Context, liveStreamID int64) (auctioncli.CurrentAuctionItem, auctioncli.StepResult) {
	f.currentAuctionCalls++
	if f.currentAuctionAfterCalls > 0 && f.currentAuctionCalls >= f.currentAuctionAfterCalls {
		item := f.currentAuction
		if item.LiveStreamID == 0 {
			item.LiveStreamID = liveStreamID
		}
		return item, auctioncli.StepResult{Step: "current_auction_by_live_stream", OK: true, RefID: item.AuctionID, StatusCode: 200}
	}
	return auctioncli.CurrentAuctionItem{}, auctioncli.StepResult{Step: "current_auction_by_live_stream", OK: true, StatusCode: 200}
}
