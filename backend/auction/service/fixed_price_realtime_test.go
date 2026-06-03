package service

import (
	"context"
	"sync"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"auction-service/model"
)

type fixedPriceEvent struct {
	typeName     string
	liveStreamID int64
	itemID       int64
	buyerID      int64
	remaining    int
	price        string
}

type fakeFixedPriceBroadcaster struct {
	mu     sync.Mutex
	events []fixedPriceEvent
}

func (f *fakeFixedPriceBroadcaster) Listed(_ context.Context, item *model.FixedPriceItem) {
	f.add(fixedPriceEvent{
		typeName:     "listed",
		liveStreamID: item.LiveStreamID,
		itemID:       item.ID,
		remaining:    item.RemainingStock,
		price:        item.Price.StringFixed(2),
	})
}

func (f *fakeFixedPriceBroadcaster) StockChanged(_ context.Context, liveStreamID, itemID int64, remaining int) {
	f.add(fixedPriceEvent{typeName: "stock", liveStreamID: liveStreamID, itemID: itemID, remaining: remaining})
}

func (f *fakeFixedPriceBroadcaster) SoldOut(_ context.Context, liveStreamID, itemID int64) {
	f.add(fixedPriceEvent{typeName: "sold_out", liveStreamID: liveStreamID, itemID: itemID})
}

func (f *fakeFixedPriceBroadcaster) Offline(_ context.Context, liveStreamID, itemID int64) {
	f.add(fixedPriceEvent{typeName: "offline", liveStreamID: liveStreamID, itemID: itemID})
}

func (f *fakeFixedPriceBroadcaster) Flair(_ context.Context, liveStreamID, itemID, buyerID int64, price decimal.Decimal) {
	f.add(fixedPriceEvent{
		typeName:     "flair",
		liveStreamID: liveStreamID,
		itemID:       itemID,
		buyerID:      buyerID,
		price:        price.StringFixed(2),
	})
}

func (f *fakeFixedPriceBroadcaster) add(e fixedPriceEvent) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, e)
}

func (f *fakeFixedPriceBroadcaster) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = nil
}

func (f *fakeFixedPriceBroadcaster) snapshot() []fixedPriceEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]fixedPriceEvent, len(f.events))
	copy(out, f.events)
	return out
}

func setupFixedPriceServiceWithBroadcaster(t *testing.T, b FixedPriceBroadcaster) *FixedPriceService {
	t.Helper()
	db := setupServiceDB(t)
	rdb := setupTestRedis(t)
	return NewFixedPriceService(
		db,
		newItemDAO(db), newPurchaseDAO(db), newBalanceDAO(db),
		NewStockGuard(rdb), NewIdemStore(rdb),
		&fakeStreamOwner{owners: nil},
		&fakeProductChecker{},
		nil,
		b,
	)
}

func TestFixedPriceServiceRealtime_ListItemBroadcastsListed(t *testing.T) {
	b := &fakeFixedPriceBroadcaster{}
	svc := setupFixedPriceServiceWithBroadcaster(t, b)

	item, err := svc.ListItem(context.Background(), ListItemReq{
		LiveStreamID: 1001, ProductID: 5001, CreatorID: 100,
		Price: decimal.NewFromInt(99), TotalStock: 10, MaxPerUser: 1,
	})
	require.NoError(t, err)

	events := b.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, "listed", events[0].typeName)
	assert.Equal(t, item.ID, events[0].itemID)
	assert.Equal(t, int64(1001), events[0].liveStreamID)
	assert.Equal(t, 10, events[0].remaining)
	assert.Equal(t, "99.00", events[0].price)
}

func TestFixedPriceServiceRealtime_PurchaseBroadcastsStockAndFlair(t *testing.T) {
	b := &fakeFixedPriceBroadcaster{}
	svc := setupFixedPriceServiceWithBroadcaster(t, b)
	ctx := context.Background()
	item := setupItem(t, svc, 5, decimal.NewFromInt(99))
	setBalance(t, svc, 100, decimal.NewFromInt(1000))
	b.reset()

	_, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: newKey()})
	require.NoError(t, err)

	events := b.snapshot()
	require.Len(t, events, 2)
	assert.Equal(t, "stock", events[0].typeName)
	assert.Equal(t, 4, events[0].remaining)
	assert.Equal(t, "flair", events[1].typeName)
	assert.Equal(t, int64(100), events[1].buyerID)
	assert.Equal(t, "99.00", events[1].price)
}

func TestFixedPriceServiceRealtime_PurchaseReplayDoesNotBroadcast(t *testing.T) {
	b := &fakeFixedPriceBroadcaster{}
	svc := setupFixedPriceServiceWithBroadcaster(t, b)
	ctx := context.Background()
	item := setupItem(t, svc, 5, decimal.NewFromInt(99))
	setBalance(t, svc, 100, decimal.NewFromInt(1000))
	key := "550e8400-e29b-41d4-a716-446655440099"

	_, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: key})
	require.NoError(t, err)
	b.reset()

	res, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: key})
	require.NoError(t, err)
	require.True(t, res.Replayed)
	assert.Empty(t, b.snapshot())
}

func TestFixedPriceServiceRealtime_LastUnitBroadcastsStockThenSoldOut(t *testing.T) {
	b := &fakeFixedPriceBroadcaster{}
	svc := setupFixedPriceServiceWithBroadcaster(t, b)
	ctx := context.Background()
	item := setupItem(t, svc, 1, decimal.NewFromInt(99))
	setBalance(t, svc, 100, decimal.NewFromInt(1000))
	b.reset()

	_, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: newKey()})
	require.NoError(t, err)

	events := b.snapshot()
	require.Len(t, events, 3)
	assert.Equal(t, "stock", events[0].typeName)
	assert.Equal(t, 0, events[0].remaining)
	assert.Equal(t, "flair", events[1].typeName)
	assert.Equal(t, "sold_out", events[2].typeName)
}

func TestFixedPriceServiceRealtime_OfflineBroadcastsOffline(t *testing.T) {
	b := &fakeFixedPriceBroadcaster{}
	svc := setupFixedPriceServiceWithBroadcaster(t, b)
	ctx := context.Background()
	item := setupItem(t, svc, 5, decimal.NewFromInt(99))
	b.reset()

	require.NoError(t, svc.Offline(ctx, item.ID, item.CreatorID))

	events := b.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, "offline", events[0].typeName)
	assert.Equal(t, item.ID, events[0].itemID)
	assert.Equal(t, int64(1001), events[0].liveStreamID)
}

func TestFixedPriceServiceRealtime_FailurePathsDoNotBroadcast(t *testing.T) {
	b := &fakeFixedPriceBroadcaster{}
	svc := setupFixedPriceServiceWithBroadcaster(t, b)
	ctx := context.Background()
	item := setupItem(t, svc, 5, decimal.NewFromInt(99))
	b.reset()

	_, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: newKey()})
	assert.ErrorIs(t, err, ErrInsufficient)

	err = svc.Offline(ctx, item.ID, 9999)
	assert.ErrorIs(t, err, ErrNotStreamOwner)

	assert.Empty(t, b.snapshot())
}
