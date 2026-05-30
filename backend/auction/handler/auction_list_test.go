package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"auction-service/client"
	"auction-service/dao"
	"auction-service/model"
)

// fakeProductClient 是 client.ProductClient 的可控替身，用于 BuildAuctionListResponse 编排测试。
type fakeProductClient struct {
	listIDs       []int64
	listErr       error
	listCalledFor int64

	batchOut       map[int64]client.ProductSummary
	batchErr       error
	batchCalledIDs []int64
}

func (f *fakeProductClient) ListProductIDsByCategory(_ context.Context, categoryID int64) ([]int64, error) {
	f.listCalledFor = categoryID
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listIDs, nil
}

func (f *fakeProductClient) BatchGetSummaries(_ context.Context, ids []int64) (map[int64]client.ProductSummary, error) {
	f.batchCalledIDs = append([]int64(nil), ids...)
	if f.batchErr != nil {
		return nil, f.batchErr
	}
	return f.batchOut, nil
}

// fakeLister 模拟 service.ListAuctionsWithFilters，断言收到的 filters。
type fakeLister struct {
	called      bool
	gotFilters  *dao.AuctionFilters
	gotPage     int
	gotPageSize int

	out      []model.Auction
	outTotal int64
	outErr   error
}

func (l *fakeLister) List(_ context.Context, f *dao.AuctionFilters, page, pageSize int) ([]model.Auction, int64, error) {
	l.called = true
	l.gotFilters = f
	l.gotPage = page
	l.gotPageSize = pageSize
	return l.out, l.outTotal, l.outErr
}

func TestBuildAuctionListResponse(t *testing.T) {
	ctx := context.Background()

	t.Run("no category_id: skip product list, still attach product summaries via batch", func(t *testing.T) {
		// 数据：两条 auction，关联两个 product
		now := time.Now()
		fl := &fakeLister{
			out: []model.Auction{
				{ID: 100, ProductID: 11, Status: model.AuctionStatusOngoing, CurrentPrice: 200, StartTime: now, EndTime: now.Add(time.Hour)},
				{ID: 101, ProductID: 22, Status: model.AuctionStatusEnded, CurrentPrice: 300, StartTime: now, EndTime: now.Add(time.Hour)},
			},
			outTotal: 2,
		}
		cid7 := int64(7)
		fp := &fakeProductClient{
			batchOut: map[int64]client.ProductSummary{
				11: {ID: 11, Name: "p1", Images: []string{"u1", "u1b"}, CategoryID: &cid7},
				22: {ID: 22, Name: "p2", Images: []string{}, CategoryID: nil},
			},
		}

		params := ListParams{Page: 1, PageSize: 20}
		items, total, err := BuildAuctionListResponse(ctx, fp, fl.List, params)
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		require.Len(t, items, 2)

		// 编排断言：未传 category_id → ListProductIDsByCategory 不被调
		assert.Equal(t, int64(0), fp.listCalledFor)
		// dao filter 中不应携带 ProductIDs
		require.NotNil(t, fl.gotFilters)
		assert.Empty(t, fl.gotFilters.ProductIDs)
		// batch 被以 [11,22] 调用（顺序不强约束，长度即可）
		assert.ElementsMatch(t, []int64{11, 22}, fp.batchCalledIDs)

		// 摘要内嵌：spec C §4.1 list 返回 image (=images[0])
		assert.Equal(t, int64(11), items[0].Product.ID)
		assert.Equal(t, "p1", items[0].Product.Name)
		assert.Equal(t, "u1", items[0].Product.Image)
		require.NotNil(t, items[0].Product.CategoryID)
		assert.Equal(t, int64(7), *items[0].Product.CategoryID)
		assert.Equal(t, "", items[1].Product.Image) // 空 images → ""
	})

	t.Run("with category_id: filter by ids before listing auctions", func(t *testing.T) {
		fl := &fakeLister{
			out:      []model.Auction{{ID: 200, ProductID: 33}},
			outTotal: 1,
		}
		fp := &fakeProductClient{
			listIDs: []int64{33, 44, 55},
			batchOut: map[int64]client.ProductSummary{
				33: {ID: 33, Name: "p3", Images: []string{"u3"}},
			},
		}
		cid := int64(7)
		params := ListParams{CategoryID: &cid, Page: 1, PageSize: 20}

		items, total, err := BuildAuctionListResponse(ctx, fp, fl.List, params)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		require.Len(t, items, 1)

		// 编排断言：先调 list 拿到 [33,44,55]，作为 dao filter 注入
		assert.Equal(t, int64(7), fp.listCalledFor)
		require.NotNil(t, fl.gotFilters)
		assert.Equal(t, []int64{33, 44, 55}, fl.gotFilters.ProductIDs)
		// batch 只对实际命中的 product_id 调用
		assert.Equal(t, []int64{33}, fp.batchCalledIDs)
	})

	t.Run("with category_id but no matching products: short-circuit empty result", func(t *testing.T) {
		fl := &fakeLister{}
		fp := &fakeProductClient{listIDs: []int64{}}
		cid := int64(7)
		params := ListParams{CategoryID: &cid, Page: 1, PageSize: 20}

		items, total, err := BuildAuctionListResponse(ctx, fp, fl.List, params)
		require.NoError(t, err)
		assert.Equal(t, int64(0), total)
		assert.Empty(t, items)
		// 不应进入 dao 查询和 batch
		assert.False(t, fl.called)
		assert.Empty(t, fp.batchCalledIDs)
	})

	t.Run("category list call fails: return error (no silent fallback)", func(t *testing.T) {
		fl := &fakeLister{}
		fp := &fakeProductClient{listErr: errors.New("product-service down")}
		cid := int64(7)
		params := ListParams{CategoryID: &cid, Page: 1, PageSize: 20}

		_, _, err := BuildAuctionListResponse(ctx, fp, fl.List, params)
		require.Error(t, err)
		assert.False(t, fl.called)
	})

	t.Run("batch summaries call fails: return error (no silent fallback)", func(t *testing.T) {
		fl := &fakeLister{
			out:      []model.Auction{{ID: 1, ProductID: 11}},
			outTotal: 1,
		}
		fp := &fakeProductClient{batchErr: errors.New("batch down")}
		params := ListParams{Page: 1, PageSize: 20}

		_, _, err := BuildAuctionListResponse(ctx, fp, fl.List, params)
		require.Error(t, err)
	})

	t.Run("auction whose product missing in batch result: product fields empty but item kept", func(t *testing.T) {
		fl := &fakeLister{
			out:      []model.Auction{{ID: 1, ProductID: 99}},
			outTotal: 1,
		}
		// batch 返回空 map：表示 product-service 未找到该 id
		fp := &fakeProductClient{batchOut: map[int64]client.ProductSummary{}}

		items, total, err := BuildAuctionListResponse(ctx, fp, fl.List, ListParams{Page: 1, PageSize: 20})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		require.Len(t, items, 1)
		assert.Equal(t, int64(99), items[0].ProductID)
		assert.Equal(t, int64(0), items[0].Product.ID)
		assert.Equal(t, "", items[0].Product.Name)
	})
}
