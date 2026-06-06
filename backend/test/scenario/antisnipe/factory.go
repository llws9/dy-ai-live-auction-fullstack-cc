// Package antisnipe 真实 AuctionFactory 实现：通过业务 SDK 创建拍品 + 竞拍规则 + 拍卖。
package antisnipe

import (
	"context"
	"fmt"
	"time"

	"test-service/client/auction"
)

// SDKAuctionFactory 通过 auction SDK 创建短倒计时拍卖
type SDKAuctionFactory struct {
	cli      *auction.Client
	sellerID int64
	duration int // 单位 秒
}

// NewSDKAuctionFactory 构造
//   - cli   : 业务 SDK 客户端
//   - seller: 创建拍卖的卖家 ID
//   - dur   : 拍卖倒计时秒数（建议 30s 用于演示）
func NewSDKAuctionFactory(cli *auction.Client, seller int64, dur int) *SDKAuctionFactory {
	if dur <= 0 {
		dur = 30
	}
	return &SDKAuctionFactory{cli: cli, sellerID: seller, duration: dur}
}

// Prepare 创建一个商品、竞拍规则和拍卖。
func (f *SDKAuctionFactory) Prepare(ctx context.Context, name string) (int64, error) {
	seller := auction.Actor{
		UserID:   f.sellerID,
		Username: fmt.Sprintf("merchant_%d", f.sellerID),
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
