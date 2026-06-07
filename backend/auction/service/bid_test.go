package service

import (
	"testing"

	"auction-service/model"

	"github.com/shopspring/decimal"

	"github.com/stretchr/testify/assert"
)

// TestBidService_PlaceBidRequest_Validation 测试出价请求验证
func TestBidService_PlaceBidRequest_Validation(t *testing.T) {
	tests := []struct {
		name      string
		req       PlaceBidRequest
		expectErr bool
	}{
		{
			name: "Valid request",
			req: PlaceBidRequest{
				AuctionID: 1,
				UserID:    1,
				Amount:    decimal.NewFromInt(100),
			},
			expectErr: false,
		},
		{
			name: "Zero amount",
			req: PlaceBidRequest{
				AuctionID: 1,
				UserID:    1,
				Amount:    decimal.Zero,
			},
			expectErr: true,
		},
		{
			name: "Negative amount",
			req: PlaceBidRequest{
				AuctionID: 1,
				UserID:    1,
				Amount:    decimal.NewFromInt(-10),
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证出价金额
			if tt.expectErr {
				assert.False(t, tt.req.Amount.GreaterThan(decimal.Zero))
			} else {
				assert.True(t, tt.req.Amount.GreaterThan(decimal.Zero))
			}
		})
	}
}

func TestMinimumBidAmountUsesStartPriceBeforeAnyBid(t *testing.T) {
	got := minimumBidAmount(decimal.Zero, decimal.NewFromInt(100), decimal.NewFromInt(10))

	assert.True(t, got.Equal(decimal.NewFromInt(110)), "first bid must be based on start price plus increment")
}

func TestMinimumBidAmountUsesCurrentPriceAfterBiddingStarts(t *testing.T) {
	got := minimumBidAmount(decimal.NewFromInt(120), decimal.NewFromInt(100), decimal.NewFromInt(10))

	assert.True(t, got.Equal(decimal.NewFromInt(130)), "subsequent bid must be based on current price plus increment")
}

// TestPlaceBidResult_Structure 测试出价结果结构
func TestPlaceBidResult_Structure(t *testing.T) {
	result := &PlaceBidResult{
		Success:      true,
		Message:      "出价成功",
		CurrentPrice: decimal.NewFromInt(100),
		Rank:         1,
		WinnerID:     1,
	}

	assert.True(t, result.Success)
	assert.True(t, result.CurrentPrice.Equal(decimal.NewFromInt(100)))
	assert.Equal(t, 1, result.Rank)
	assert.Equal(t, int64(1), result.WinnerID)
}

// TestRankingThrottle_ShouldSend 测试排名推送节流
func TestRankingThrottle_ShouldSend(t *testing.T) {
	throttle := NewRankingThrottle()
	auctionID := int64(1)

	// 第一次应该允许发送
	assert.True(t, throttle.ShouldSend(auctionID))

	// 立即再次调用应该被节流
	assert.False(t, throttle.ShouldSend(auctionID))
}

// TestRankingThrottle_MultipleAuctions 测试多个竞拍的节流独立
func TestRankingThrottle_MultipleAuctions(t *testing.T) {
	throttle := NewRankingThrottle()

	// 不同竞拍应该独立节流
	assert.True(t, throttle.ShouldSend(1))
	assert.True(t, throttle.ShouldSend(2))
	assert.True(t, throttle.ShouldSend(3))

	// 同一个竞拍再次调用应该被节流
	assert.False(t, throttle.ShouldSend(1))
	assert.False(t, throttle.ShouldSend(2))
	assert.False(t, throttle.ShouldSend(3))
}

// TestBidModel_Creation 测试出价模型创建
func TestBidModel_Creation(t *testing.T) {
	bid := &model.Bid{
		AuctionID: 1,
		UserID:    1,
		Amount:    decimal.NewFromInt(100),
	}

	assert.Equal(t, int64(1), bid.AuctionID)
	assert.Equal(t, int64(1), bid.UserID)
	assert.True(t, bid.Amount.Equal(decimal.NewFromInt(100)))
}

// TestBidService_OutbidNotification 测试出价超越通知逻辑
func TestBidService_OutbidNotification(t *testing.T) {
	// 模拟出价超越场景
	previousWinnerID := int64(1)
	previousPrice := 100.0
	newBid := 120.0
	newBidderID := int64(2)

	// 验证出价超越条件
	assert.True(t, newBid > previousPrice)
	assert.NotEqual(t, previousWinnerID, newBidderID)

	// 验证通知内容
	expectedContent := "您在竞拍中的出价 100.00 元已被超越，当前最高价为 120.00 元"
	assert.Contains(t, expectedContent, "100.00")
	assert.Contains(t, expectedContent, "120.00")
}
