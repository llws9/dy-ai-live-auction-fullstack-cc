package service

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"auction-service/dao"
	"auction-service/model"
)

// 一口价业务错误。handler 层负责映射为对外错误码与 HTTP 状态。
var (
	ErrInvalidParam    = errors.New("invalid param")
	ErrNotStreamOwner  = errors.New("not stream owner")
	ErrProductNotFound = errors.New("product not found")
	ErrNotOnSale       = errors.New("fixed price item not on sale")
	ErrSoldOut         = errors.New("fixed price item sold out")
	ErrAlreadyBought   = errors.New("user already bought this fixed price item")
	ErrInsufficient    = errors.New("insufficient balance")
)

const (
	maxTotalStock = 10000
	cleanupDelay  = 5 * time.Second
)

var (
	minFixedPriceAmount = decimal.NewFromInt(1).Shift(-2)
	maxFixedPriceAmount = decimal.NewFromInt(9999999999).Shift(-2)
)

// StreamOwnerChecker 校验某用户是否为指定直播间的主播（创建者）。
type StreamOwnerChecker interface {
	IsOwner(ctx context.Context, liveStreamID, userID int64) (bool, error)
}

// ProductChecker 校验商品是否存在。
type ProductChecker interface {
	Exists(ctx context.Context, productID int64) (bool, error)
}

// BalanceDeducter 在事务内条件扣减用户余额（affected==0 表示余额不足）。
type BalanceDeducter interface {
	DeductWithTx(ctx context.Context, tx *gorm.DB, userID int64, amount decimal.Decimal) (int64, error)
}

// Clock 抽象延时调度，便于测试用 fake 时钟驱动异步清理。
type Clock interface {
	AfterFunc(d time.Duration, f func())
}

// realClock 基于 time.AfterFunc 的生产实现。
type realClock struct{}

func (realClock) AfterFunc(d time.Duration, f func()) { time.AfterFunc(d, f) }

// FixedPriceService 一口价业务编排（方案③ purchase 自成闭环：
// 抢购在 auction 单库单事务内完成 扣余额 + 写购买记录，不跨服务建单、不依赖 outbox）。
type FixedPriceService struct {
	db          *gorm.DB
	items       *dao.FixedPriceItemDAO
	purchases   *dao.FixedPricePurchaseDAO
	balance     BalanceDeducter
	stock       *StockGuard
	idem        *IdemStore
	streams     StreamOwnerChecker
	products    ProductChecker
	clk         Clock
	broadcaster FixedPriceBroadcaster
}

// NewFixedPriceService 装配一口价 service。clk 传 nil 时使用 realClock。
func NewFixedPriceService(
	db *gorm.DB,
	items *dao.FixedPriceItemDAO,
	purchases *dao.FixedPricePurchaseDAO,
	balance BalanceDeducter,
	stock *StockGuard,
	idem *IdemStore,
	streams StreamOwnerChecker,
	products ProductChecker,
	clk Clock,
	broadcaster FixedPriceBroadcaster,
) *FixedPriceService {
	if clk == nil {
		clk = realClock{}
	}
	if broadcaster == nil {
		broadcaster = noopFixedPriceBroadcaster{}
	}
	return &FixedPriceService{
		db:          db,
		items:       items,
		purchases:   purchases,
		balance:     balance,
		stock:       stock,
		idem:        idem,
		streams:     streams,
		products:    products,
		clk:         clk,
		broadcaster: broadcaster,
	}
}

// ListItemReq 上架请求。
type ListItemReq struct {
	LiveStreamID int64
	ProductID    int64
	CreatorID    int64
	Price        decimal.Decimal
	TotalStock   int
	MaxPerUser   int
}

// ListItem 上架一口价商品：校验参数、主播归属、商品存在，落库并初始化 Redis 库存。
func (s *FixedPriceService) ListItem(ctx context.Context, r ListItemReq) (*model.FixedPriceItem, error) {
	if !validFixedPriceAmount(r.Price) || r.TotalStock <= 0 || r.TotalStock > maxTotalStock {
		return nil, ErrInvalidParam
	}
	isOwner, err := s.streams.IsOwner(ctx, r.LiveStreamID, r.CreatorID)
	if err != nil {
		return nil, err
	}
	if !isOwner {
		return nil, ErrNotStreamOwner
	}
	exists, err := s.products.Exists(ctx, r.ProductID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrProductNotFound
	}
	if r.MaxPerUser <= 0 {
		r.MaxPerUser = 1
	}

	item := &model.FixedPriceItem{
		LiveStreamID:   r.LiveStreamID,
		ProductID:      r.ProductID,
		CreatorID:      r.CreatorID,
		Price:          r.Price,
		TotalStock:     r.TotalStock,
		RemainingStock: r.TotalStock,
		MaxPerUser:     r.MaxPerUser,
		Status:         model.FixedPriceStatusOnSale,
	}
	if err := s.items.Create(ctx, item); err != nil {
		return nil, err
	}
	if err := s.stock.Init(ctx, item.ID, r.TotalStock); err != nil {
		_ = s.items.UpdateStatus(ctx, item.ID, model.FixedPriceStatusOffline)
		return nil, err
	}
	s.broadcaster.Listed(ctx, item)
	return item, nil
}

func validFixedPriceAmount(price decimal.Decimal) bool {
	if price.LessThan(minFixedPriceAmount) || price.GreaterThan(maxFixedPriceAmount) {
		return false
	}
	return price.Equal(price.Round(2))
}

// PurchaseReq 抢购请求。IdemKey 必须为 UUID v4 形态。
type PurchaseReq struct {
	ItemID  int64
	UserID  int64
	IdemKey string
}

// PurchaseResult 抢购结果。方案③：PurchaseID 即购买凭证（fixed_price_purchases.id），不跨服务建单。
type PurchaseResult struct {
	PurchaseID     int64
	ItemID         int64
	Price          decimal.Decimal
	RemainingStock int
	Replayed       bool
}

// ListLiveItemsReq 查询直播间一口价商品列表。
type ListLiveItemsReq struct {
	LiveStreamID int64
}

// LiveFixedPriceItem 是公开直播间一口价列表的单项结果。
type LiveFixedPriceItem struct {
	Item           *model.FixedPriceItem
	RemainingStock int
}

// ListByLiveStream 返回指定直播间的在售一口价商品，库存优先使用 Redis 权威值。
func (s *FixedPriceService) ListByLiveStream(ctx context.Context, r ListLiveItemsReq) ([]*LiveFixedPriceItem, error) {
	if r.LiveStreamID <= 0 {
		return nil, ErrInvalidParam
	}
	items, err := s.items.ListByLiveStreamID(ctx, r.LiveStreamID, []model.FixedPriceStatus{model.FixedPriceStatusOnSale})
	if err != nil {
		return nil, err
	}
	out := make([]*LiveFixedPriceItem, 0, len(items))
	for _, item := range items {
		remaining := item.RemainingStock
		if live, e := s.stock.Remaining(ctx, item.ID); e == nil {
			remaining = live
		}
		out = append(out, &LiveFixedPriceItem{Item: item, RemainingStock: remaining})
	}
	return out, nil
}

// Purchase 抢购一口价商品（方案③ purchase 自成闭环）。
//
// 链路：幂等校验 → 幂等命中复用 → 状态预检 → Lua 原子预扣库存 →
// auction 单库单事务（扣余额 + 写购买记录）→ 失败 Saga 补偿（回补 Redis 库存）→
// 成功后持久化幂等键 → 末件标记售罄。
func (s *FixedPriceService) Purchase(ctx context.Context, r PurchaseReq) (*PurchaseResult, error) {
	if !s.idem.IsValidKey(r.IdemKey) {
		return nil, ErrInvalidParam
	}

	// 1. 幂等命中：复用已记录的 purchase ID，不再扣库存/余额。
	if purchaseID, hit, err := s.idem.GetOrInit(ctx, r.UserID, r.ItemID, r.IdemKey, 0); err != nil {
		return nil, err
	} else if hit {
		item, err := s.items.GetByID(ctx, r.ItemID)
		if err != nil {
			return nil, err
		}
		rem, _ := s.stock.Remaining(ctx, r.ItemID)
		return &PurchaseResult{
			PurchaseID: purchaseID, ItemID: r.ItemID, Price: item.Price,
			RemainingStock: rem, Replayed: true,
		}, nil
	}

	// 2. 状态预检：售罄返回更具体的 ErrSoldOut，其余非在售（已下架）返回 ErrNotOnSale。
	item, err := s.items.GetByID(ctx, r.ItemID)
	if err != nil {
		return nil, err
	}
	switch item.Status {
	case model.FixedPriceStatusOnSale:
		// 继续
	case model.FixedPriceStatusSoldOut:
		return nil, ErrSoldOut
	default:
		return nil, ErrNotOnSale
	}

	// 3. Lua 原子预扣库存。
	res, err := s.stock.TryAcquire(ctx, r.ItemID, r.UserID)
	if err != nil {
		return nil, err
	}
	switch res {
	case StockResultUninitialized:
		return nil, ErrNotOnSale
	case StockResultSoldOut:
		return nil, ErrSoldOut
	case StockResultAlreadyBought:
		return s.replayExistingPurchase(ctx, item, r.UserID, r.IdemKey)
	}
	rem, err := s.stock.Remaining(ctx, r.ItemID)
	if err != nil {
		_ = s.stock.Compensate(ctx, r.ItemID, r.UserID)
		return nil, err
	}

	// 4. 单库单事务：扣余额 + 写购买记录。
	purchase := &model.FixedPricePurchase{
		ItemID: item.ID, UserID: r.UserID, IdempotencyKey: r.IdemKey, Price: item.Price,
	}
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		affected, e := s.balance.DeductWithTx(ctx, tx, r.UserID, item.Price)
		if e != nil {
			return e
		}
		if affected == 0 {
			return ErrInsufficient
		}
		if e := s.purchases.InsertWithTx(ctx, tx, purchase); e != nil {
			return e
		}
		return nil
	})

	// 5. 事务失败 → Saga 补偿，回补 Redis 库存与 bought 集合。
	if txErr != nil {
		_ = s.stock.Compensate(ctx, r.ItemID, r.UserID)
		if errors.Is(txErr, dao.ErrAlreadyBought) {
			return s.replayExistingPurchase(ctx, item, r.UserID, r.IdemKey)
		}
		return nil, txErr
	}

	// 6. 持久化幂等键（存 purchase ID）。
	_ = s.idem.Persist(ctx, r.UserID, r.ItemID, r.IdemKey, purchase.ID)
	_ = s.items.DecrementRemainingStock(ctx, r.ItemID, rem)

	// 7. 实时广播。库存 DB 兜底写与广播均为 best-effort，不影响核心购买结果。
	s.broadcaster.StockChanged(ctx, item.LiveStreamID, item.ID, rem)
	s.broadcaster.Flair(ctx, item.LiveStreamID, item.ID, r.UserID, item.Price)
	if rem == 0 {
		if err := s.items.UpdateStatus(ctx, r.ItemID, model.FixedPriceStatusSoldOut); err == nil {
			s.broadcaster.SoldOut(ctx, item.LiveStreamID, item.ID)
		}
	}

	return &PurchaseResult{
		PurchaseID: purchase.ID, ItemID: r.ItemID, Price: item.Price,
		RemainingStock: rem, Replayed: false,
	}, nil
}

func (s *FixedPriceService) replayExistingPurchase(ctx context.Context, item *model.FixedPriceItem, userID int64, idemKey string) (*PurchaseResult, error) {
	var purchase *model.FixedPricePurchase
	var err error
	for i := 0; i < 20; i++ {
		purchase, err = s.purchases.GetByItemAndUser(ctx, item.ID, userID)
		if err == nil {
			break
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		return nil, ErrAlreadyBought
	}
	if purchase.IdempotencyKey != idemKey {
		return nil, ErrAlreadyBought
	}
	_ = s.idem.Persist(ctx, userID, item.ID, idemKey, purchase.ID)
	rem, _ := s.stock.Remaining(ctx, item.ID)
	return &PurchaseResult{
		PurchaseID: purchase.ID, ItemID: item.ID, Price: purchase.Price,
		RemainingStock: rem, Replayed: true,
	}, nil
}

// Offline 下架一口价商品：仅主播可下架，软标记状态后延时清理 Redis 库存。
func (s *FixedPriceService) Offline(ctx context.Context, itemID, userID int64) error {
	item, err := s.items.GetByID(ctx, itemID)
	if err != nil {
		return err
	}
	if item.CreatorID != userID {
		return ErrNotStreamOwner
	}
	if err := s.items.UpdateStatus(ctx, itemID, model.FixedPriceStatusOffline); err != nil {
		return err
	}
	s.broadcaster.Offline(ctx, item.LiveStreamID, item.ID)
	s.scheduleCleanup(itemID)
	return nil
}

// scheduleCleanup 在 cleanupDelay 后清理 Redis 库存与购买集合，给在途请求留补偿窗口。
func (s *FixedPriceService) scheduleCleanup(itemID int64) {
	s.clk.AfterFunc(cleanupDelay, func() {
		_ = s.stock.Cleanup(context.Background(), itemID)
	})
}

// GetItem 读取一口价商品（详情/错误码拼装用）。
func (s *FixedPriceService) GetItem(ctx context.Context, itemID int64) (*model.FixedPriceItem, error) {
	return s.items.GetByID(ctx, itemID)
}

// RemainingStock 读取 Redis 权威剩余库存。
func (s *FixedPriceService) RemainingStock(ctx context.Context, itemID int64) (int, error) {
	return s.stock.Remaining(ctx, itemID)
}

// GetMyPurchase 查询当前用户对某商品的购买记录（无跨域，spec §5.2 i_bought）。
func (s *FixedPriceService) GetMyPurchase(ctx context.Context, itemID, userID int64) (*model.FixedPricePurchase, error) {
	return s.purchases.GetByItemAndUser(ctx, itemID, userID)
}
