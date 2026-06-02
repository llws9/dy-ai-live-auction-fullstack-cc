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
