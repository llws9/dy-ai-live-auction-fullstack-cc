package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"auction-service/dao"
	"auction-service/model"
)

// --- fakes ---

// fakeStreamOwner 以 (liveStreamID -> ownerUserID) 映射判定归属；未登记的直播间视为归属任意人。
type fakeStreamOwner struct {
	owners map[int64]int64 // liveStreamID -> ownerUserID；nil 表示放行所有
}

func (f *fakeStreamOwner) IsOwner(_ context.Context, liveStreamID, userID int64) (bool, error) {
	if f.owners == nil {
		return true, nil
	}
	owner, ok := f.owners[liveStreamID]
	if !ok {
		return false, nil
	}
	return owner == userID, nil
}

// fakeProductChecker 默认所有商品存在。
type fakeProductChecker struct {
	missing map[int64]bool
}

func (f *fakeProductChecker) Exists(_ context.Context, productID int64) (bool, error) {
	if f.missing[productID] {
		return false, nil
	}
	return true, nil
}

type fakeAuctionChecker struct {
	auctions map[int64]*model.Auction
	missing  map[int64]bool
}

func (f *fakeAuctionChecker) GetByID(_ context.Context, id int64) (*model.Auction, error) {
	if f == nil || f.missing[id] {
		return nil, gorm.ErrRecordNotFound
	}
	if auction, ok := f.auctions[id]; ok {
		return auction, nil
	}
	liveStreamID := int64(1001)
	creatorID := int64(100)
	return &model.Auction{
		ID:           id,
		LiveStreamID: &liveStreamID,
		CreatorID:    &creatorID,
		Status:       model.AuctionStatusOngoing,
	}, nil
}

// fakeClock 手动推进的时钟，用于驱动异步清理。
type fakeClock struct {
	mu      sync.Mutex
	now     time.Time
	pending []*fakeTimer
}

type fakeTimer struct {
	at time.Time
	f  func()
}

func newFakeClock() *fakeClock { return &fakeClock{now: time.Unix(0, 0)} }

func (c *fakeClock) AfterFunc(d time.Duration, f func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pending = append(c.pending, &fakeTimer{at: c.now.Add(d), f: f})
}

// Advance 推进时钟并同步触发到期回调。
func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	var due []*fakeTimer
	var rest []*fakeTimer
	for _, t := range c.pending {
		if !t.at.After(c.now) {
			due = append(due, t)
		} else {
			rest = append(rest, t)
		}
	}
	c.pending = rest
	c.mu.Unlock()
	for _, t := range due {
		t.f()
	}
}

// --- service builders ---

var svcDBCounter int64

func setupServiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	svcDBCounter++
	dsn := "file:svc_test_" + itoa(svcDBCounter) + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.FixedPriceItem{},
		&model.FixedPricePurchase{},
		&model.UserBalance{},
	))
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func newItemDAO(db *gorm.DB) *dao.FixedPriceItemDAO         { return dao.NewFixedPriceItemDAO(db) }
func newPurchaseDAO(db *gorm.DB) *dao.FixedPricePurchaseDAO { return dao.NewFixedPricePurchaseDAO(db) }
func newBalanceDAO(db *gorm.DB) *dao.UserBalanceDAO         { return dao.NewUserBalanceDAO(db) }

// setupFixedPriceService 构造一个全放行（owner/product）的 service，使用真实时钟。
func setupFixedPriceService(t *testing.T) *FixedPriceService {
	return setupFixedPriceServiceWithClock(t, nil)
}

// setupFixedPriceServiceWithClock 允许注入 fake 时钟。
func setupFixedPriceServiceWithClock(t *testing.T, clk Clock) *FixedPriceService {
	t.Helper()
	db := setupServiceDB(t)
	rdb := setupTestRedis(t)
	return NewFixedPriceService(
		db,
		dao.NewFixedPriceItemDAO(db),
		dao.NewFixedPricePurchaseDAO(db),
		dao.NewUserBalanceDAO(db),
		NewStockGuard(rdb),
		NewIdemStore(rdb),
		&fakeStreamOwner{owners: nil},
		&fakeProductChecker{},
		&fakeAuctionChecker{},
		clk,
		nil,
	)
}

// setupFixedPriceServiceWithStream 构造一个限定直播间归属的 service。
func setupFixedPriceServiceWithStream(t *testing.T, liveStreamID, ownerUserID int64) *FixedPriceService {
	t.Helper()
	db := setupServiceDB(t)
	rdb := setupTestRedis(t)
	return NewFixedPriceService(
		db,
		dao.NewFixedPriceItemDAO(db),
		dao.NewFixedPricePurchaseDAO(db),
		dao.NewUserBalanceDAO(db),
		NewStockGuard(rdb),
		NewIdemStore(rdb),
		&fakeStreamOwner{owners: map[int64]int64{liveStreamID: ownerUserID}},
		&fakeProductChecker{},
		&fakeAuctionChecker{},
		nil,
		nil,
	)
}

// setupItem 上架一件商品供测试用，CreatorID 固定 100、LiveStreamID 固定 1001。
func setupItem(t *testing.T, svc *FixedPriceService, stock int, price decimal.Decimal) *model.FixedPriceItem {
	t.Helper()
	item, err := svc.ListItem(context.Background(), ListItemReq{
		AuctionID: 7001, LiveStreamID: 1001, ProductID: 5001, CreatorID: 100,
		Price: price, TotalStock: stock, MaxPerUser: 1,
	})
	require.NoError(t, err)
	return item
}

// setBalance 直接写入用户余额。
func setBalance(t *testing.T, svc *FixedPriceService, userID int64, amount decimal.Decimal) {
	t.Helper()
	require.NoError(t, svc.db.Create(&model.UserBalance{
		UserID:          userID,
		AvailableAmount: amount,
		Currency:        "CNY",
	}).Error)
}

// newKey 返回一个合法 UUID v4 形态的幂等键。
var keyCounter int64

func newKey() string {
	keyCounter++
	// 形如 550e8400-e29b-41d4-a716-XXXXXXXXXXXX，末段递增确保唯一。
	suffix := itoa(446655440000 + keyCounter)
	for len(suffix) < 12 {
		suffix = "0" + suffix
	}
	return "550e8400-e29b-41d4-a716-" + suffix
}

// eventually 在超时内轮询断言条件成立。
func eventually(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}
