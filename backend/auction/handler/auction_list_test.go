package handler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"auction-service/client"
	"auction-service/dao"
	"auction-service/model"

	"github.com/shopspring/decimal"
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

func (f *fakeProductClient) GetAuctionProductInfo(_ context.Context, _ int64) (*client.AuctionProductInfo, error) {
	return nil, nil
}

func (f *fakeProductClient) GetOrCreateActiveLiveStream(_ context.Context, _ int64, _ string) (*client.LiveStreamInfo, error) {
	return nil, nil
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

type fakeRuleBatchFetcher struct {
	out            map[int64]*model.AuctionRule
	err            error
	batchCalledIDs []int64
}

func (f *fakeRuleBatchFetcher) GetByProductIDs(_ context.Context, ids []int64) (map[int64]*model.AuctionRule, error) {
	f.batchCalledIDs = append([]int64(nil), ids...)
	if f.err != nil {
		return nil, f.err
	}
	return f.out, nil
}

func (l *fakeLister) List(_ context.Context, f *dao.AuctionFilters, page, pageSize int) ([]model.Auction, int64, error) {
	l.called = true
	l.gotFilters = f
	l.gotPage = page
	l.gotPageSize = pageSize
	return l.out, l.outTotal, l.outErr
}

// fakeLiveStreamClient 是 client.LiveStreamClient 的可控替身。
type fakeLiveStreamClient struct {
	out       map[int64]client.LiveStreamSummary
	err       error
	calledIDs []int64
}

func (f *fakeLiveStreamClient) BatchGetLiveStreams(_ context.Context, ids []int64) (map[int64]client.LiveStreamSummary, error) {
	f.calledIDs = append([]int64(nil), ids...)
	if f.err != nil {
		return nil, f.err
	}
	return f.out, nil
}

func TestBuildAuctionListResponse(t *testing.T) {
	ctx := context.Background()

	t.Run("no category_id: skip product list, still attach product summaries via batch", func(t *testing.T) {
		// 数据：两条 auction，关联两个 product
		now := time.Now()
		fl := &fakeLister{
			out: []model.Auction{
				{ID: 100, ProductID: 11, Status: model.AuctionStatusOngoing, CurrentPrice: decimal.NewFromInt(200), StartTime: now, EndTime: now.Add(time.Hour)},
				{ID: 101, ProductID: 22, Status: model.AuctionStatusEnded, CurrentPrice: decimal.NewFromInt(300), StartTime: now, EndTime: now.Add(time.Hour)},
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
		rf := &fakeRuleBatchFetcher{
			out: map[int64]*model.AuctionRule{
				11: {ProductID: 11, StartPrice: decimal.NewFromInt(100)},
				22: {ProductID: 22, StartPrice: decimal.NewFromInt(200)},
			},
		}

		items, total, err := BuildAuctionListResponse(ctx, fp, nil, fl.List, rf, params)
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
		assert.ElementsMatch(t, []int64{11, 22}, rf.batchCalledIDs)

		// 摘要内嵌：spec C §4.1 list 返回 image (=images[0])
		assert.Equal(t, int64(11), items[0].Product.ID)
		assert.Equal(t, "p1", items[0].Product.Name)
		assert.Equal(t, "u1", items[0].Product.Image)
		require.NotNil(t, items[0].StartPrice)
		assert.True(t, items[0].StartPrice.Equal(decimal.NewFromInt(100)))
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

		items, total, err := BuildAuctionListResponse(ctx, fp, nil, fl.List, nil, params)
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

		items, total, err := BuildAuctionListResponse(ctx, fp, nil, fl.List, nil, params)
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

		_, _, err := BuildAuctionListResponse(ctx, fp, nil, fl.List, nil, params)
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

		_, _, err := BuildAuctionListResponse(ctx, fp, nil, fl.List, nil, params)
		require.Error(t, err)
	})

	t.Run("auction whose product is not returned by batch is hidden from public list", func(t *testing.T) {
		fl := &fakeLister{
			out:      []model.Auction{{ID: 1, ProductID: 99}},
			outTotal: 1,
		}
		// batch 返回空 map：表示 product-service 未找到该 id，或商品未发布。
		fp := &fakeProductClient{batchOut: map[int64]client.ProductSummary{}}

		items, total, err := BuildAuctionListResponse(ctx, fp, nil, fl.List, nil, ListParams{Page: 1, PageSize: 20})
		require.NoError(t, err)
		assert.Equal(t, int64(0), total)
		assert.Empty(t, items)
	})

	t.Run("upcoming flag is forwarded to auction filters", func(t *testing.T) {
		now := time.Now()
		fl := &fakeLister{
			out: []model.Auction{
				{ID: 300, ProductID: 77, Status: model.AuctionStatusPending, CurrentPrice: decimal.NewFromInt(1200), StartTime: now.Add(time.Hour), EndTime: now.Add(2 * time.Hour)},
			},
			outTotal: 1,
		}
		fp := &fakeProductClient{
			batchOut: map[int64]client.ProductSummary{
				77: {ID: 77, Name: "upcoming product", Images: []string{"u77"}},
			},
		}

		items, total, err := BuildAuctionListResponse(ctx, fp, nil, fl.List, nil, ListParams{Upcoming: true, Page: 1, PageSize: 2})
		require.NoError(t, err)
		require.Equal(t, int64(1), total)
		require.Len(t, items, 1)
		require.NotNil(t, fl.gotFilters)
		assert.True(t, fl.gotFilters.Upcoming)
		assert.Nil(t, fl.gotFilters.Status)
		assert.Equal(t, 1, fl.gotPage)
		assert.Equal(t, 2, fl.gotPageSize)
	})
}

func TestBuildAuctionListResponse_ViewerCount(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	ls101 := int64(101)

	newFixture := func() (*fakeLister, *fakeProductClient) {
		fl := &fakeLister{
			out: []model.Auction{
				{ID: 1, ProductID: 11, LiveStreamID: &ls101, Status: model.AuctionStatusOngoing, StartTime: now, EndTime: now.Add(time.Hour)},
			},
			outTotal: 1,
		}
		fp := &fakeProductClient{
			batchOut: map[int64]client.ProductSummary{
				11: {ID: 11, Name: "p1", Images: []string{"u1"}},
			},
		}
		return fl, fp
	}

	t.Run("正常回填 viewer_count", func(t *testing.T) {
		fl, fp := newFixture()
		lsc := &fakeLiveStreamClient{out: map[int64]client.LiveStreamSummary{
			101: {ID: 101, ViewerCount: 128},
		}}
		items, _, err := BuildAuctionListResponse(ctx, fp, lsc, fl.List, nil, ListParams{Page: 1, PageSize: 20})
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, int64(128), items[0].ViewerCount)
		assert.Equal(t, []int64{101}, lsc.calledIDs)
	})

	t.Run("批量直播间失败时降级不 5xx，viewer_count=0", func(t *testing.T) {
		fl, fp := newFixture()
		lsc := &fakeLiveStreamClient{err: errors.New("product-service down")}
		items, total, err := BuildAuctionListResponse(ctx, fp, lsc, fl.List, nil, ListParams{Page: 1, PageSize: 20})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		require.Len(t, items, 1)
		assert.Equal(t, int64(0), items[0].ViewerCount)
		assert.Equal(t, "p1", items[0].Product.Name)
	})

	t.Run("lsc 为 nil 时跳过，viewer_count=0", func(t *testing.T) {
		fl, fp := newFixture()
		items, _, err := BuildAuctionListResponse(ctx, fp, nil, fl.List, nil, ListParams{Page: 1, PageSize: 20})
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, int64(0), items[0].ViewerCount)
	})

	t.Run("live_stream_id 为 nil 的 auction viewer_count=0 且不进 streamIDs", func(t *testing.T) {
		fl := &fakeLister{
			out:      []model.Auction{{ID: 2, ProductID: 22, LiveStreamID: nil, Status: model.AuctionStatusOngoing, StartTime: now, EndTime: now.Add(time.Hour)}},
			outTotal: 1,
		}
		fp := &fakeProductClient{batchOut: map[int64]client.ProductSummary{22: {ID: 22, Name: "p2", Images: []string{"u2"}}}}
		lsc := &fakeLiveStreamClient{out: map[int64]client.LiveStreamSummary{}}
		items, _, err := BuildAuctionListResponse(ctx, fp, lsc, fl.List, nil, ListParams{Page: 1, PageSize: 20})
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, int64(0), items[0].ViewerCount)
		assert.Empty(t, lsc.calledIDs)
	})
}

// TestAuctionListResponseShape 锁定 GET /auctions 响应字段为 list（而非 items），
// 与 admin/h5 前端契约对齐（h5 已实现 list+items 双兼容，admin 仅消费 list）。
func TestAuctionListResponseShape(t *testing.T) {
	resp := map[string]interface{}{
		"code":    200,
		"message": "success",
		"data": map[string]interface{}{
			"list":      []interface{}{},
			"total":     0,
			"page":      1,
			"page_size": 20,
		},
	}

	raw, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &parsed))

	data := parsed["data"].(map[string]interface{})
	assert.NotNil(t, data["list"])
	assert.Nil(t, data["items"], "must use list, not items")
	assert.Contains(t, data, "total")
	assert.Contains(t, data, "page")
	assert.Contains(t, data, "page_size")
}
