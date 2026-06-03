package service

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"auction-service/model"
)

func TestFixedPriceService_List_ValidatesAndCreates(t *testing.T) {
	svc := setupFixedPriceService(t)
	ctx := context.Background()
	item, err := svc.ListItem(ctx, ListItemReq{
		LiveStreamID: 1001, ProductID: 5001, CreatorID: 100,
		Price: decimal.NewFromFloat(99), TotalStock: 50, MaxPerUser: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, 50, item.RemainingStock)
	remain, err := svc.stock.Remaining(ctx, item.ID)
	require.NoError(t, err)
	assert.Equal(t, 50, remain)
}

func TestFixedPriceService_List_RejectsInvalidPrice(t *testing.T) {
	svc := setupFixedPriceService(t)
	_, err := svc.ListItem(context.Background(), ListItemReq{
		LiveStreamID: 1, ProductID: 1, CreatorID: 1,
		Price: decimal.Zero, TotalStock: 10,
	})
	assert.ErrorIs(t, err, ErrInvalidParam)
}

func TestFixedPriceService_List_RejectsPriceOutsideTwoCentScale(t *testing.T) {
	svc := setupFixedPriceService(t)
	tests := []decimal.Decimal{
		decimal.NewFromFloat(0.001),
		decimal.NewFromFloat(99.999),
	}
	for _, price := range tests {
		_, err := svc.ListItem(context.Background(), ListItemReq{
			LiveStreamID: 1, ProductID: 1, CreatorID: 1,
			Price: price, TotalStock: 10,
		})
		assert.ErrorIs(t, err, ErrInvalidParam)
	}
}

func TestFixedPriceService_List_RejectsExcessiveStock(t *testing.T) {
	svc := setupFixedPriceService(t)
	_, err := svc.ListItem(context.Background(), ListItemReq{
		LiveStreamID: 1, ProductID: 1, CreatorID: 1,
		Price: decimal.NewFromInt(10), TotalStock: 10001,
	})
	assert.ErrorIs(t, err, ErrInvalidParam)
}

func TestFixedPriceService_List_RejectsNonOwner(t *testing.T) {
	svc := setupFixedPriceServiceWithStream(t, 1001, 100) // owner=100
	_, err := svc.ListItem(context.Background(), ListItemReq{
		LiveStreamID: 1001, ProductID: 5001, CreatorID: 999, // not owner
		Price: decimal.NewFromInt(99), TotalStock: 10,
	})
	assert.ErrorIs(t, err, ErrNotStreamOwner)
}

func TestFixedPriceService_List_RejectsMissingProduct(t *testing.T) {
	db := setupServiceDB(t)
	rdb := setupTestRedis(t)
	svc := NewFixedPriceService(
		db,
		newItemDAO(db), newPurchaseDAO(db), newBalanceDAO(db),
		NewStockGuard(rdb), NewIdemStore(rdb),
		&fakeStreamOwner{owners: nil},
		&fakeProductChecker{missing: map[int64]bool{5001: true}},
		nil,
		nil,
	)
	_, err := svc.ListItem(context.Background(), ListItemReq{
		LiveStreamID: 1001, ProductID: 5001, CreatorID: 100,
		Price: decimal.NewFromInt(99), TotalStock: 10,
	})
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestFixedPriceService_ListByLiveStream_ReturnsOnSaleItemsWithRedisStock(t *testing.T) {
	svc := setupFixedPriceService(t)
	ctx := context.Background()
	item := setupItem(t, svc, 5, decimal.NewFromInt(99))
	res, err := svc.stock.TryAcquire(ctx, item.ID, 100)
	require.NoError(t, err)
	require.Equal(t, StockResultSuccess, res)

	items, err := svc.ListByLiveStream(ctx, ListLiveItemsReq{LiveStreamID: item.LiveStreamID})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, item.ID, items[0].Item.ID)
	assert.Equal(t, 4, items[0].RemainingStock)
}

func TestFixedPriceService_ListAllByLiveStream_ReturnsAllStatusesForAdmin(t *testing.T) {
	svc := setupFixedPriceService(t)
	ctx := context.Background()
	onSale := setupItem(t, svc, 5, decimal.NewFromInt(99))
	soldOut := setupItem(t, svc, 3, decimal.NewFromInt(88))
	offline := setupItem(t, svc, 2, decimal.NewFromInt(77))
	require.NoError(t, svc.items.UpdateStatus(ctx, soldOut.ID, model.FixedPriceStatusSoldOut))
	require.NoError(t, svc.items.UpdateStatus(ctx, offline.ID, model.FixedPriceStatusOffline))

	items, err := svc.ListAllByLiveStream(ctx, ListLiveItemsReq{LiveStreamID: onSale.LiveStreamID})
	require.NoError(t, err)
	require.Len(t, items, 3)

	statuses := make(map[model.FixedPriceStatus]bool)
	for _, item := range items {
		statuses[item.Item.Status] = true
	}
	assert.True(t, statuses[model.FixedPriceStatusOnSale])
	assert.True(t, statuses[model.FixedPriceStatusSoldOut])
	assert.True(t, statuses[model.FixedPriceStatusOffline])
}

// ---- T7 抢购 ----

func TestPurchase_HappyPath(t *testing.T) {
	svc := setupFixedPriceService(t)
	ctx := context.Background()
	item := setupItem(t, svc, 5, decimal.NewFromInt(99))
	setBalance(t, svc, 100, decimal.NewFromInt(1000))

	res, err := svc.Purchase(ctx, PurchaseReq{
		ItemID: item.ID, UserID: 100, IdemKey: "550e8400-e29b-41d4-a716-446655440000",
	})
	require.NoError(t, err)
	assert.NotZero(t, res.PurchaseID)
	assert.Equal(t, 4, res.RemainingStock)
	assert.False(t, res.Replayed)

	// 余额扣减落库
	avail, _, _, hit, err := newBalanceDAO(svc.db).GetByUserID(ctx, 100)
	require.NoError(t, err)
	require.True(t, hit)
	assert.Equal(t, "901.00", avail.StringFixed(2))
}

func TestPurchase_SoldOut(t *testing.T) {
	svc := setupFixedPriceService(t)
	ctx := context.Background()
	item := setupItem(t, svc, 1, decimal.NewFromInt(10))
	setBalance(t, svc, 100, decimal.NewFromInt(100))
	setBalance(t, svc, 200, decimal.NewFromInt(100))

	_, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: newKey()})
	require.NoError(t, err)
	_, err = svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 200, IdemKey: newKey()})
	assert.ErrorIs(t, err, ErrSoldOut)
}

func TestPurchase_AlreadyBought(t *testing.T) {
	svc := setupFixedPriceService(t)
	ctx := context.Background()
	item := setupItem(t, svc, 5, decimal.NewFromInt(10))
	setBalance(t, svc, 100, decimal.NewFromInt(100))

	_, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: newKey()})
	require.NoError(t, err)
	_, err = svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: newKey()})
	assert.ErrorIs(t, err, ErrAlreadyBought)
}

func TestPurchase_InsufficientBalance_TriggersCompensation(t *testing.T) {
	svc := setupFixedPriceService(t)
	ctx := context.Background()
	item := setupItem(t, svc, 5, decimal.NewFromInt(99))
	setBalance(t, svc, 100, decimal.NewFromInt(50))

	_, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: newKey()})
	assert.ErrorIs(t, err, ErrInsufficient)

	// Saga 补偿：库存回补、bought 集合移除
	remain, _ := svc.stock.Remaining(ctx, item.ID)
	assert.Equal(t, 5, remain)
	bought, _ := svc.stock.rdb.SIsMember(ctx, boughtKey(item.ID), int64(100)).Result()
	assert.False(t, bought)
}

func TestPurchase_IdempotentReplay(t *testing.T) {
	svc := setupFixedPriceService(t)
	ctx := context.Background()
	item := setupItem(t, svc, 5, decimal.NewFromInt(99))
	setBalance(t, svc, 100, decimal.NewFromInt(1000))
	key := "550e8400-e29b-41d4-a716-446655440001"

	res1, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: key})
	require.NoError(t, err)
	res2, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: key})
	require.NoError(t, err)
	assert.Equal(t, res1.PurchaseID, res2.PurchaseID)
	assert.True(t, res2.Replayed)

	// 仅扣一次库存
	remain, _ := svc.stock.Remaining(ctx, item.ID)
	assert.Equal(t, 4, remain)
	// 仅扣一次余额
	avail, _, _, _, _ := newBalanceDAO(svc.db).GetByUserID(ctx, 100)
	assert.Equal(t, "901.00", avail.StringFixed(2))
}

func TestPurchase_ReplaysFromPurchaseWhenIdemCacheMissing(t *testing.T) {
	svc := setupFixedPriceService(t)
	ctx := context.Background()
	item := setupItem(t, svc, 5, decimal.NewFromInt(99))
	setBalance(t, svc, 100, decimal.NewFromInt(1000))
	key := "550e8400-e29b-41d4-a716-446655440101"

	res1, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: key})
	require.NoError(t, err)
	require.NoError(t, svc.idem.rdb.Del(ctx, idemKey(100, item.ID, key)).Err())

	res2, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: key})
	require.NoError(t, err)
	assert.Equal(t, res1.PurchaseID, res2.PurchaseID)
	assert.True(t, res2.Replayed)

	remain, err := svc.stock.Remaining(ctx, item.ID)
	require.NoError(t, err)
	assert.Equal(t, 4, remain)
	avail, _, _, _, _ := newBalanceDAO(svc.db).GetByUserID(ctx, 100)
	assert.Equal(t, "901.00", avail.StringFixed(2))
}

func TestPurchase_UpdatesDBRemainingStockForFallback(t *testing.T) {
	svc := setupFixedPriceService(t)
	ctx := context.Background()
	item := setupItem(t, svc, 5, decimal.NewFromInt(99))
	setBalance(t, svc, 100, decimal.NewFromInt(1000))

	_, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: newKey()})
	require.NoError(t, err)

	got, err := svc.items.GetByID(ctx, item.ID)
	require.NoError(t, err)
	assert.Equal(t, 4, got.RemainingStock)
}

func TestPurchase_NotOnSale(t *testing.T) {
	svc := setupFixedPriceService(t)
	ctx := context.Background()
	item := setupItem(t, svc, 5, decimal.NewFromInt(10))
	setBalance(t, svc, 100, decimal.NewFromInt(100))
	require.NoError(t, svc.Offline(ctx, item.ID, item.CreatorID))

	_, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: newKey()})
	assert.ErrorIs(t, err, ErrNotOnSale)
}

func TestPurchase_LastUnitMarksSoldOut(t *testing.T) {
	svc := setupFixedPriceService(t)
	ctx := context.Background()
	item := setupItem(t, svc, 1, decimal.NewFromInt(10))
	setBalance(t, svc, 100, decimal.NewFromInt(100))

	res, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: newKey()})
	require.NoError(t, err)
	assert.Equal(t, 0, res.RemainingStock)

	got, err := svc.items.GetByID(ctx, item.ID)
	require.NoError(t, err)
	assert.Equal(t, model.FixedPriceStatusSoldOut, got.Status)
}

func TestPurchase_Concurrent_NoOversell(t *testing.T) {
	svc := setupFixedPriceService(t)
	ctx := context.Background()
	item := setupItem(t, svc, 50, decimal.NewFromInt(1))

	for i := 0; i < 100; i++ {
		setBalance(t, svc, int64(1000+i), decimal.NewFromInt(10))
	}

	var wg sync.WaitGroup
	var success int64
	for i := 0; i < 100; i++ {
		wg.Add(1)
		userID := int64(1000 + i)
		go func() {
			defer wg.Done()
			_, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: userID, IdemKey: newKeyFor(userID)})
			if err == nil {
				atomic.AddInt64(&success, 1)
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, int64(50), success)
	remain, _ := svc.stock.Remaining(ctx, item.ID)
	assert.Equal(t, 0, remain)
}

// newKeyFor 为指定用户生成确定且合法的 UUID v4 形态幂等键（并发安全，无共享计数）。
func newKeyFor(userID int64) string {
	suffix := strconv.FormatInt(446600000000+userID, 10)
	for len(suffix) < 12 {
		suffix = "0" + suffix
	}
	return "550e8400-e29b-41d4-a716-" + suffix
}

// ---- T8 下架 ----

func TestOffline_OwnerMarksOnly(t *testing.T) {
	svc := setupFixedPriceService(t)
	ctx := context.Background()
	item := setupItem(t, svc, 10, decimal.NewFromInt(99))

	require.NoError(t, svc.Offline(ctx, item.ID, item.CreatorID))

	got, err := svc.items.GetByID(ctx, item.ID)
	require.NoError(t, err)
	assert.Equal(t, model.FixedPriceStatusOffline, got.Status)
	// 真实时钟下 5s 内不会清理，库存仍在
	rem, err := svc.stock.Remaining(ctx, item.ID)
	require.NoError(t, err)
	assert.Equal(t, 10, rem)
}

func TestOffline_NonOwner(t *testing.T) {
	svc := setupFixedPriceService(t)
	ctx := context.Background()
	item := setupItem(t, svc, 10, decimal.NewFromInt(99))

	err := svc.Offline(ctx, item.ID, 9999)
	assert.ErrorIs(t, err, ErrNotStreamOwner)
}

func TestOffline_AsyncCleanupAfterDelay(t *testing.T) {
	clk := newFakeClock()
	svc := setupFixedPriceServiceWithClock(t, clk)
	ctx := context.Background()
	item := setupItem(t, svc, 10, decimal.NewFromInt(99))

	require.NoError(t, svc.Offline(ctx, item.ID, item.CreatorID))
	// 推进前库存仍在
	_, err := svc.stock.rdb.Get(ctx, stockKey(item.ID)).Result()
	require.NoError(t, err)

	clk.Advance(6 * time.Second)
	eventually(t, func() bool {
		_, err := svc.stock.rdb.Get(ctx, stockKey(item.ID)).Result()
		return err == redis.Nil
	})
}
