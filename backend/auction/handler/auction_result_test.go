package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"auction-service/client"
	"auction-service/model"

	"github.com/shopspring/decimal"
)

// fakeAuctionFetcher 模拟 service.GetAuction。
type fakeAuctionFetcher struct {
	out *model.Auction
	err error
}

func (f *fakeAuctionFetcher) Get(_ context.Context, _ int64) (*model.Auction, error) {
	return f.out, f.err
}

type fakeWinnerBidFetcher struct {
	out       *model.Bid
	err       error
	calledIDs []int64
}

func (f *fakeWinnerBidFetcher) GetWinnerBid(_ context.Context, auctionID int64) (*model.Bid, error) {
	f.calledIDs = append(f.calledIDs, auctionID)
	return f.out, f.err
}

type fakeResultUserFetcher struct {
	out       map[int64]*model.User
	err       error
	calledIDs []int64
}

func (f *fakeResultUserFetcher) GetByIDs(_ context.Context, ids []int64) (map[int64]*model.User, error) {
	f.calledIDs = append(f.calledIDs, ids...)
	return f.out, f.err
}

func TestBuildAuctionResultResponse(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	winnerID := int64(88)

	t.Run("returns full structure with embedded product", func(t *testing.T) {
		af := &fakeAuctionFetcher{
			out: &model.Auction{
				ID:           1001,
				ProductID:    5001,
				Status:       model.AuctionStatusEnded,
				CurrentPrice: decimal.NewFromInt(9200),
				WinnerID:     &winnerID,
				StartTime:    now,
				EndTime:      now.Add(30 * time.Minute),
				DelayUsed:    30,
			},
		}
		fp := &fakeProductClient{
			batchOut: map[int64]client.ProductSummary{
				5001: {
					ID:     5001,
					Name:   "和田玉手镯",
					Images: []string{"https://cdn.example.com/p/5001/0.jpg", "https://cdn.example.com/p/5001/1.jpg"},
				},
			},
		}
		wb := &fakeWinnerBidFetcher{
			out: &model.Bid{
				ID:        7001,
				AuctionID: 1001,
				UserID:    winnerID,
				Amount:    decimal.NewFromInt(9200),
				CreatedAt: now.Add(29 * time.Minute),
			},
		}
		uf := &fakeResultUserFetcher{
			out: map[int64]*model.User{
				winnerID: {ID: winnerID, Name: "买家A"},
			},
		}

		got, err := BuildAuctionResultResponse(ctx, fp, af.Get, wb.GetWinnerBid, uf.GetByIDs, 1001)
		require.NoError(t, err)

		// 已有字段保留
		assert.Equal(t, int64(1001), got.AuctionID)
		assert.Equal(t, int64(5001), got.ProductID)
		assert.Equal(t, model.AuctionStatusEnded, got.Status)
		assert.Equal(t, float64(9200), got.FinalPrice)
		require.NotNil(t, got.WinnerID)
		assert.Equal(t, int64(88), *got.WinnerID)
		assert.Equal(t, 30, got.DelayUsed)

		// 新增字段：won_bid 返回中标出价详情，供 H5 结果页展示中标人和出价时间。
		require.NotNil(t, got.WonBid)
		assert.Equal(t, int64(7001), got.WonBid.ID)
		assert.Equal(t, int64(88), got.WonBid.UserID)
		assert.Equal(t, "买家A", got.WonBid.UserName)
		assert.Equal(t, float64(9200), got.WonBid.Amount)
		assert.Equal(t, now.Add(29*time.Minute), got.WonBid.CreatedAt)

		// 新增字段：product 含完整 images 数组（与 list 仅 image 不同）
		require.NotNil(t, got.Product)
		assert.Equal(t, int64(5001), got.Product.ID)
		assert.Equal(t, "和田玉手镯", got.Product.Name)
		assert.Equal(t, []string{
			"https://cdn.example.com/p/5001/0.jpg",
			"https://cdn.example.com/p/5001/1.jpg",
		}, got.Product.Images)

		// 编排断言：用 [5001] 调 batch
		assert.Equal(t, []int64{5001}, fp.batchCalledIDs)
	})

	t.Run("auction not found: bubble error", func(t *testing.T) {
		af := &fakeAuctionFetcher{err: errors.New("not found")}
		fp := &fakeProductClient{}

		_, err := BuildAuctionResultResponse(ctx, fp, af.Get, nil, nil, 9999)
		require.Error(t, err)
		// product client 不应被调用
		assert.Empty(t, fp.batchCalledIDs)
	})

	t.Run("product-service fails: soft fallback with product=nil", func(t *testing.T) {
		// 用户决策：result 接口 product 调用失败时软降级，
		// 核心字段照常返回，product=null。
		af := &fakeAuctionFetcher{
			out: &model.Auction{
				ID:           1001,
				ProductID:    5001,
				Status:       model.AuctionStatusEnded,
				CurrentPrice: decimal.NewFromInt(9200),
			},
		}
		fp := &fakeProductClient{batchErr: errors.New("product-service down")}

		got, err := BuildAuctionResultResponse(ctx, fp, af.Get, nil, nil, 1001)
		require.NoError(t, err)
		assert.Nil(t, got.Product)
		// 核心字段仍正确
		assert.Nil(t, got.WonBid)
		assert.Equal(t, float64(9200), got.FinalPrice)
	})

	t.Run("product missing in batch result: product=nil", func(t *testing.T) {
		af := &fakeAuctionFetcher{
			out: &model.Auction{
				ID:           1001,
				ProductID:    5001,
				Status:       model.AuctionStatusEnded,
				CurrentPrice: decimal.NewFromInt(9200),
			},
		}
		fp := &fakeProductClient{batchOut: map[int64]client.ProductSummary{}}

		got, err := BuildAuctionResultResponse(ctx, fp, af.Get, nil, nil, 1001)
		require.NoError(t, err)
		assert.Nil(t, got.Product)
	})

	t.Run("nil product client: still works, product=nil", func(t *testing.T) {
		// productClient 未注入时，不应崩溃；product=null 软降级。
		af := &fakeAuctionFetcher{
			out: &model.Auction{ID: 1, ProductID: 2, CurrentPrice: decimal.NewFromInt(100)},
		}

		got, err := BuildAuctionResultResponse(ctx, nil, af.Get, nil, nil, 1)
		require.NoError(t, err)
		assert.Nil(t, got.Product)
		assert.Nil(t, got.WonBid)
	})
}
