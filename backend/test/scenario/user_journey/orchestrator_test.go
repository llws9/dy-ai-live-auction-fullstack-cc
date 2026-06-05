package user_journey

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"test-service/client/auction"
)

func TestRunHappyPathProducesEvidenceReport(t *testing.T) {
	ctx := context.Background()
	biz := newFakeBiz()
	internal := &fakeInternalClient{}
	rec := &fakeSeedRecorder{}
	emitter := &fakeEmitter{}

	report, err := New(biz, internal, rec, Config{TestID: "tj_1"}).Run(ctx, emitter)
	require.NoError(t, err)

	assert.True(t, report.AllOK)
	assert.Equal(t, "tj_1", report.TestRunID)
	assert.Equal(t, int64(101), report.ProductID)
	assert.Equal(t, int64(201), report.LiveStreamID)
	assert.Equal(t, int64(301), report.AuctionID)
	assert.Equal(t, int64(401), report.FixedPriceItemID)
	assert.Equal(t, int64(501), report.OrderID)
	assert.Equal(t, "0.00", report.BalanceBefore)
	assert.Equal(t, "900.00", report.BalanceAfter)
	assert.Equal(t, int64(1), report.StockBefore)
	assert.Equal(t, int64(0), report.StockAfter)
	assertStepOrder(t, report.Steps, []string{
		"prepare",
		"enter_live",
		"reminder",
		"auction_bid",
		"sky_lamp",
		"fixed_price_purchase",
		"verify",
	})
	assert.Equal(t, []string{"product:101", "live_stream:201", "auction:301", "fixed_price_item:401", "order:501"}, rec.added)
	assert.False(t, rec.deleteCalled, "user_journey must keep evidence by default")
	assert.Equal(t, "verify", emitter.lastStep())
}

func TestPrepareFailsClosedWhenTopUpFails(t *testing.T) {
	ctx := context.Background()
	biz := newFakeBiz()
	internal := &fakeInternalClient{topUpErr: errors.New("auction internal down")}
	rec := &fakeSeedRecorder{}

	report, err := New(biz, internal, rec, Config{TestID: "tj_fail"}).Run(ctx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prepare top_up_balance failed")
	assert.False(t, report.AllOK)
	assertStepOrder(t, report.Steps, []string{"prepare"})
}

func TestPrepareSkipsCleanupAndStillRecordsSeedRefs(t *testing.T) {
	ctx := context.Background()
	rec := &fakeSeedRecorder{}

	_, err := New(newFakeBiz(), &fakeInternalClient{}, rec, Config{TestID: "tj_keep"}).Run(ctx, nil)
	require.NoError(t, err)

	assert.False(t, rec.deleteCalled)
	assert.Contains(t, rec.added, "product:101")
	assert.Contains(t, rec.added, "live_stream:201")
	assert.Contains(t, rec.added, "auction:301")
	assert.Contains(t, rec.added, "fixed_price_item:401")
}

func TestRunDeletesSeedRefsWhenKeepEvidenceFalse(t *testing.T) {
	ctx := context.Background()
	rec := &fakeSeedRecorder{}
	keepEvidence := false

	_, err := New(newFakeBiz(), &fakeInternalClient{}, rec, Config{
		TestID:       "tj_cleanup",
		KeepEvidence: &keepEvidence,
	}).Run(ctx, nil)
	require.NoError(t, err)

	assert.True(t, rec.deleteCalled)
	assert.Equal(t, "tj_cleanup", rec.deletedTestID)
}

func TestCleanupFailureDoesNotOverrideBusinessSuccess(t *testing.T) {
	ctx := context.Background()
	rec := &fakeSeedRecorder{deleteErr: errors.New("cleanup failed")}
	keepEvidence := false

	report, err := New(newFakeBiz(), &fakeInternalClient{}, rec, Config{
		TestID:       "tj_cleanup_warn",
		KeepEvidence: &keepEvidence,
	}).Run(ctx, nil)
	require.NoError(t, err)

	assert.True(t, report.AllOK)
	assert.Contains(t, report.Warnings, "cleanup failed: cleanup failed")
	assert.Empty(t, report.Error)
}

func TestReminderStepUsesFollowAndFollowStatusOnly(t *testing.T) {
	ctx := context.Background()
	biz := newFakeBiz()

	_, err := New(biz, &fakeInternalClient{}, &fakeSeedRecorder{}, Config{TestID: "tj_reminder"}).Run(ctx, nil)
	require.NoError(t, err)

	assert.Contains(t, biz.calls, "follow_live_stream")
	assert.Contains(t, biz.calls, "get_follow_status")
	assert.NotContains(t, biz.calls, "toggle_notification")
	assert.NotContains(t, biz.calls, "pending_reminder")
}

type fakeBiz struct {
	calls []string
}

func newFakeBiz() *fakeBiz { return &fakeBiz{calls: make([]string, 0, 16)} }

func (f *fakeBiz) call(name string) { f.calls = append(f.calls, name) }

func (f *fakeBiz) CreateProductAs(_ context.Context, _ auction.Actor, _ auction.CreateProductReq) auction.StepResult {
	f.call("create_product")
	return okStep("create_product", 101)
}

func (f *fakeBiz) CreateLiveStream(_ context.Context, _ auction.Actor, _ auction.CreateLiveStreamReq) auction.StepResult {
	f.call("create_live_stream")
	return okStep("create_live_stream", 201)
}

func (f *fakeBiz) CreateAuctionAs(_ context.Context, _ auction.Actor, _ auction.CreateAuctionReq) auction.StepResult {
	f.call("create_auction")
	return okStep("create_auction", 301)
}

func (f *fakeBiz) CreateFixedPriceItem(_ context.Context, _ auction.Actor, _ auction.CreateFixedPriceItemReq) auction.StepResult {
	f.call("create_fixed_price_item")
	return okStep("create_fixed_price_item", 401)
}

func (f *fakeBiz) StartLive(_ context.Context, _ auction.Actor, liveStreamID int64) auction.StepResult {
	f.call("start_live")
	return okStep("start_live", liveStreamID)
}

func (f *fakeBiz) GetLiveStream(_ context.Context, _ auction.Actor, liveStreamID int64) (auction.LiveStream, auction.StepResult) {
	f.call("get_live_stream")
	return auction.LiveStream{ID: liveStreamID, Status: "ongoing"}, okStep("get_live_stream", liveStreamID)
}

func (f *fakeBiz) ListFixedPriceItemsByLiveStream(_ context.Context, _ auction.Actor, liveStreamID int64) ([]auction.FixedPriceItem, auction.StepResult) {
	f.call("list_fixed_price_items")
	return []auction.FixedPriceItem{{ID: 401, LiveStreamID: liveStreamID, Stock: 1, RemainingStock: 1}}, okStep("list_fixed_price_items", 401)
}

func (f *fakeBiz) FollowLiveStream(_ context.Context, _ auction.Actor, liveStreamID int64) auction.StepResult {
	f.call("follow_live_stream")
	return okStep("follow_live_stream", liveStreamID)
}

func (f *fakeBiz) GetFollowStatus(_ context.Context, _ auction.Actor, liveStreamID int64) (bool, auction.StepResult) {
	f.call("get_follow_status")
	return true, okStep("get_follow_status", liveStreamID)
}

func (f *fakeBiz) PlaceBid(_ context.Context, _ int64, auctionID int64, _ float64) auction.StepResult {
	f.call("place_bid")
	return okStep("place_bid", auctionID)
}

func (f *fakeBiz) SubscribeSkyLamp(_ context.Context, _ int64, auctionID int64) auction.StepResult {
	f.call("subscribe_sky_lamp")
	return okStep("subscribe_sky_lamp", 601)
}

func (f *fakeBiz) PurchaseFixedPriceItem(_ context.Context, _ auction.Actor, _ int64, _ string) (int64, auction.StepResult) {
	f.call("purchase_fixed_price_item")
	return 501, okStep("purchase_fixed_price_item", 501)
}

func (f *fakeBiz) GetMyFixedPricePurchase(_ context.Context, _ auction.Actor, itemID int64) (auction.FixedPricePurchase, auction.StepResult) {
	f.call("get_my_fixed_price_purchase")
	return auction.FixedPricePurchase{OrderID: 501, ItemID: itemID}, okStep("get_my_fixed_price_purchase", 501)
}

func (f *fakeBiz) FindOrdersByAuction(_ context.Context, _ int64, auctionID int64) ([]auction.Order, auction.StepResult) {
	f.call("find_orders")
	return []auction.Order{{ID: 501, AuctionID: auctionID}}, okStep("find_orders", 501)
}

func (f *fakeBiz) GetUserBalance(_ context.Context, _ auction.Actor) (string, auction.StepResult) {
	f.call("get_user_balance")
	if countCalls(f.calls, "get_user_balance") == 1 {
		return "0.00", okStep("get_user_balance", 0)
	}
	return "900.00", okStep("get_user_balance", 0)
}

type fakeInternalClient struct {
	topUpErr error
}

func (f *fakeInternalClient) TopUpUserBalance(_ context.Context, _ int64, _ string) (string, auction.StepResult) {
	if f.topUpErr != nil {
		return "", auction.StepResult{Step: "top_up_balance", Message: f.topUpErr.Error(), Err: f.topUpErr}
	}
	return "1000.00", okStep("top_up_balance", 0)
}

type fakeSeedRecorder struct {
	added         []string
	deleteCalled  bool
	deletedTestID string
	deleteErr     error
}

func (f *fakeSeedRecorder) Add(_ context.Context, _ string, kind string, refID int64) error {
	f.added = append(f.added, kind+":"+itoa(refID))
	return nil
}

func (f *fakeSeedRecorder) DeleteByTestID(_ context.Context, testID string) error {
	f.deleteCalled = true
	f.deletedTestID = testID
	return f.deleteErr
}

type fakeEmitter struct {
	steps []string
}

func (f *fakeEmitter) Emit(_ int, step string, _ map[string]any) {
	f.steps = append(f.steps, step)
}

func (f *fakeEmitter) lastStep() string {
	if len(f.steps) == 0 {
		return ""
	}
	return f.steps[len(f.steps)-1]
}

func okStep(step string, refID int64) auction.StepResult {
	return auction.StepResult{Step: step, OK: true, RefID: refID, StatusCode: 200}
}

func assertStepOrder(t *testing.T, got []auction.StepResult, want []string) {
	t.Helper()
	require.Len(t, got, len(want))
	for i := range want {
		assert.Equal(t, want[i], got[i].Step)
	}
}

func countCalls(calls []string, name string) int {
	n := 0
	for _, call := range calls {
		if call == name {
			n++
		}
	}
	return n
}

func itoa(v int64) string {
	switch v {
	case 101:
		return "101"
	case 201:
		return "201"
	case 301:
		return "301"
	case 401:
		return "401"
	case 501:
		return "501"
	default:
		return "0"
	}
}
