// Package antisnipe 真实 AuctionFactory 实现：通过业务 SDK 创建拍品 + 竞拍规则 + 拍卖。
package antisnipe

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"test-service/client/auction"
)

// SDKAuctionFactory 通过 auction SDK 创建短倒计时拍卖
type SDKAuctionFactory struct {
	cli        *auction.Client
	duration   int // 单位 秒
	nextSeller int64
}

// NewSDKAuctionFactory 构造
//   - cli   : 业务 SDK 客户端
//   - seller: 创建拍卖的卖家 ID
//   - dur   : 拍卖倒计时秒数（建议 30s 用于演示）
func NewSDKAuctionFactory(cli *auction.Client, seller int64, dur int) *SDKAuctionFactory {
	if dur <= 0 {
		dur = 30
	}
	seed := seller*1_000_000 + time.Now().UnixNano()%1_000_000
	return &SDKAuctionFactory{cli: cli, duration: dur, nextSeller: seed}
}

// Prepare 创建一个商品、竞拍规则和拍卖。
// 每次调用必须创建独立商品：auction-service 对同一 product_id 只允许一条 Pending/Ongoing/Delayed 活跃竞拍。
func (f *SDKAuctionFactory) Prepare(ctx context.Context, name string) (int64, error) {
	sellerID := atomic.AddInt64(&f.nextSeller, 1)
	seller := auction.Actor{
		UserID:   sellerID,
		Username: fmt.Sprintf("merchant_%d", sellerID),
		Role:     auction.RoleMerchant,
	}

	prod := f.cli.CreateProductAs(ctx, seller, auction.CreateProductReq{
		Name:        fmt.Sprintf("AntiSnipe %s %d", name, time.Now().UnixNano()),
		Description: "anti-snipe scenario auto-generated",
		Status:      1,
	})
	if !prod.OK {
		return 0, fmt.Errorf("create_product: %s", prod.Message)
	}

	rule := f.cli.CreateAuctionRule(ctx, seller, prod.RefID, auction.CreateAuctionRuleReq{
		StartPrice:         100,
		Increment:          10,
		Duration:           f.duration,
		TriggerDelayBefore: 5,
		DelayDuration:      5,
		MaxDelayTime:       30,
	})
	if !rule.OK {
		return 0, fmt.Errorf("create_auction_rule: %s", rule.Message)
	}

	au := f.cli.CreateAuctionAs(ctx, seller, auction.CreateAuctionReq{
		ProductID:  prod.RefID,
		StartPrice: 100,
		Increment:  10,
		Duration:   f.duration,
	})
	if !au.OK {
		return 0, fmt.Errorf("create_auction: %s", au.Message)
	}
	if started := f.cli.WaitAuctionStarted(ctx, au.RefID, 100*time.Millisecond, 5*time.Second); !started.OK {
		return 0, fmt.Errorf("wait_auction_started: %s", started.Message)
	}
	return au.RefID, nil
}

// Cleanup no-op（拍卖到时自动结束；商品保留供历史展示）
func (f *SDKAuctionFactory) Cleanup(ctx context.Context, auctionID int64) error {
	return nil
}
