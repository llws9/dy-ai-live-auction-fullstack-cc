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
	db        *gorm.DB
	items     *dao.FixedPriceItemDAO
	purchases *dao.FixedPricePurchaseDAO
	balance   BalanceDeducter
	stock     *StockGuard
	idem      *IdemStore
	streams   StreamOwnerChecker
	products  ProductChecker
	clk       Clock
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
) *FixedPriceService {
	if clk == nil {
		clk = realClock{}
	}
	return &FixedPriceService{
		db:        db,
		items:     items,
		purchases: purchases,
		balance:   balance,
		stock:     stock,
		idem:      idem,
		streams:   streams,
		products:  products,
		clk:       clk,
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
	if r.Price.LessThanOrEqual(decimal.Zero) || r.TotalStock <= 0 || r.TotalStock > maxTotalStock {
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
		return nil, err
	}
	return item, nil
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
		return nil, ErrAlreadyBought
	}

	// 4. 单库单事务：扣余额 + 写购买记录。
	purchase := &model.FixedPricePurchase{
		ItemID: item.ID, UserID: r.UserID, Price: item.Price,
	}
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		affected, e := s.balance.DeductWithTx(ctx, tx, r.UserID, item.Price)
		if e != nil {
			return e
		}
		if affected == 0 {
			return ErrInsufficient
		}
		return s.purchases.InsertWithTx(ctx, tx, purchase)
	})

	// 5. 事务失败 → Saga 补偿，回补 Redis 库存与 bought 集合。
	if txErr != nil {
		_ = s.stock.Compensate(ctx, r.ItemID, r.UserID)
		return nil, txErr
	}

	// 6. 持久化幂等键（存 purchase ID）。
	_ = s.idem.Persist(ctx, r.UserID, r.ItemID, r.IdemKey, purchase.ID)

	// 7. 末件标记售罄（best-effort）。
	rem, _ := s.stock.Remaining(ctx, r.ItemID)
	if rem == 0 {
		_ = s.items.UpdateStatus(ctx, r.ItemID, model.FixedPriceStatusSoldOut)
	}

	return &PurchaseResult{
		PurchaseID: purchase.ID, ItemID: r.ItemID, Price: item.Price,
		RemainingStock: rem, Replayed: false,
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
