# 首页普通竞拍卡片「真实快照观看人数」Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 H5 首页进行中的普通竞拍卡片封面图右下角，展示来自直播间的真实快照观看人数（Redis 优先、DB 兜底）。

**Architecture:** 后端契约打通——product-service 内部批量直播间接口回填 `viewer_count` → auction-service client 透传 → `BuildAuctionListResponse` 按 `live_stream_id` 批量回填（失败降级不 5xx）→ H5 类型扩展 + 仅进行中卡片渲染 pill。viewer_count 为直播间维度，前端按 `auction.status` 决定展示。

**Tech Stack:** Go (Hertz/GORM/testify, sqlite in-memory 单测)、React + Vite + Jest/RTL。

**设计依据:** [2026-06-10-h5-home-auction-viewer-count-design.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/superpowers/specs/2026-06-10-h5-home-auction-viewer-count-design.md)

**关键约束:**
- 仅 viewer_count 这一路下游失败降级（log Warn + 填 0），商品摘要失败仍维持原 5xx。
- 后端 batch 接口不做 status 过滤（纯数据语义），过滤交前端。
- 前端显示双条件：`statusInfo.live && viewerCount > 0`。

---

## Task 1: product-service 内部批量接口回填 viewer_count

**Files:**
- Modify: `backend/product/handler/internal.go`
- Modify: `backend/product/main.go:162`
- Test: `backend/product/handler/internal_test.go`

- [ ] **Step 1: 写失败测试**

在 `backend/product/handler/internal_test.go` 末尾追加。注意：`newInternalHandlerWithSeed` 当前用 `NewInternalHandler(svc, nil)`（2 参），本 Task 会把它改成 3 参，故新测试直接用即将变更的签名。

```go
func TestInternalHandler_BatchLiveStreams_ViewerCountRedisFirst(t *testing.T) {
	h := newInternalHandlerWithSeedAndViewers(t, func(db *gorm.DB) {
		require.NoError(t, db.Create(&model.LiveStream{
			ID: 101, Name: "room-a", Status: 1, CreatorID: 9, ViewerCount: 19,
		}).Error)
	}, service.StaticLiveViewerCounter{101: 42})

	body, _ := json.Marshal(map[string]interface{}{"ids": []int64{101}})
	c := app.NewContext(0)
	c.Request.SetBody(body)
	c.Request.Header.SetMethod("POST")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	h.BatchLiveStreams(context.Background(), c)

	require.Equal(t, 200, c.Response.StatusCode())
	var resp struct {
		Data struct {
			Items []struct {
				ID          int64 `json:"id"`
				ViewerCount int64 `json:"viewer_count"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &resp))
	require.Len(t, resp.Data.Items, 1)
	assert.Equal(t, int64(42), resp.Data.Items[0].ViewerCount) // Redis 优先于 DB(19)
}

func TestInternalHandler_BatchLiveStreams_ViewerCountDBFallback(t *testing.T) {
	h := newInternalHandlerWithSeedAndViewers(t, func(db *gorm.DB) {
		require.NoError(t, db.Create(&model.LiveStream{
			ID: 102, Name: "room-b", Status: 1, CreatorID: 9, ViewerCount: 7,
		}).Error)
	}, service.StaticLiveViewerCounter{}) // Redis 计数为 0

	body, _ := json.Marshal(map[string]interface{}{"ids": []int64{102}})
	c := app.NewContext(0)
	c.Request.SetBody(body)
	c.Request.Header.SetMethod("POST")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	h.BatchLiveStreams(context.Background(), c)

	require.Equal(t, 200, c.Response.StatusCode())
	var resp struct {
		Data struct {
			Items []struct {
				ViewerCount int64 `json:"viewer_count"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &resp))
	require.Len(t, resp.Data.Items, 1)
	assert.Equal(t, int64(7), resp.Data.Items[0].ViewerCount) // DB 兜底
}
```

并在文件中新增构造辅助（紧跟 `newInternalHandlerWithSeed` 之后）：

```go
// newInternalHandlerWithSeedAndViewers 在 newInternalHandlerWithSeed 基础上注入 viewerCounter。
func newInternalHandlerWithSeedAndViewers(t *testing.T, seed func(db *gorm.DB), viewers service.LiveViewerCounter) *InternalHandler {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Product{}, &model.Category{}, &model.AuctionRule{}, &model.LiveStream{}))
	db.Exec("DELETE FROM products")
	db.Exec("DELETE FROM categories")
	db.Exec("DELETE FROM auction_rules")
	db.Exec("DELETE FROM live_streams")
	if seed != nil {
		seed(db)
	}
	svc := service.NewProductService(dao.NewProductDAO(db), dao.NewAuctionRuleDAO(db), dao.NewLiveStreamDAO(db))
	lsSvc := service.NewLiveStreamServiceWithMetrics(dao.NewLiveStreamDAO(db), viewers)
	return NewInternalHandler(svc, dao.NewLiveStreamDAO(db), lsSvc)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend/product && go test ./handler/ -run TestInternalHandler_BatchLiveStreams_ViewerCount -v`
Expected: 编译失败（`NewInternalHandler` 只接受 2 参 / `liveStreamSummary` 无 `ViewerCount` 字段）。

- [ ] **Step 3: 改 internal.go——加字段、注入 counter、回填**

在 `backend/product/handler/internal.go`：

3a. `liveStreamSummary` 加字段：

```go
type liveStreamSummary struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	CoverImage  string `json:"cover_image"`
	Status      int    `json:"status"`
	CreatorID   int64  `json:"creator_id"`
	ViewerCount int64  `json:"viewer_count"`
}
```

3b. 新增最小接口 + 结构体字段 + 构造参数（替换原 `InternalHandler` 定义与 `NewInternalHandler`）：

```go
// liveViewerCounter 抽象 LiveStreamService.ViewerCountForLiveStream，便于 handler 单测注入 fake。
type liveViewerCounter interface {
	ViewerCountForLiveStream(ctx context.Context, ls *model.LiveStream) int64
}

type InternalHandler struct {
	productService *service.ProductService
	liveStreamDAO  liveStreamBatchProvider
	viewerCounter  liveViewerCounter
}

func NewInternalHandler(productService *service.ProductService, liveStreamDAO liveStreamBatchProvider, viewerCounter liveViewerCounter) *InternalHandler {
	return &InternalHandler{
		productService: productService,
		liveStreamDAO:  liveStreamDAO,
		viewerCounter:  viewerCounter,
	}
}
```

3c. `BatchLiveStreams` 组装时回填（`ls` 为 `*model.LiveStream`，与现有循环一致）。把 append 块改为：

```go
		viewerCount := int64(0)
		if h.viewerCounter != nil {
			viewerCount = h.viewerCounter.ViewerCountForLiveStream(ctx, ls)
		}
		summaries = append(summaries, liveStreamSummary{
			ID:          ls.ID,
			Name:        ls.Name,
			CoverImage:  ls.CoverImage,
			Status:      int(ls.Status),
			CreatorID:   ls.CreatorID,
			ViewerCount: viewerCount,
		})
```

> 注：`GetOrCreateActiveLiveStream` 返回的 `liveStreamSummary` 不填 `ViewerCount`（留零值），与本需求无关。

- [ ] **Step 4: 改 main.go 接线**

`backend/product/main.go:162`，把：

```go
	internalHandler := handler.NewInternalHandler(productService, liveStreamDAO)
```

改为（`liveStreamService` 已在第 126 行构造，实现了 `ViewerCountForLiveStream`）：

```go
	internalHandler := handler.NewInternalHandler(productService, liveStreamDAO, liveStreamService)
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd backend/product && go test ./handler/ -run TestInternalHandler -v`
Expected: 全部 PASS（含原有 BatchLiveStreams 用例不回归）。

- [ ] **Step 6: 全量编译 + 包测试**

Run: `cd backend/product && go build ./... && go test ./handler/ ./service/`
Expected: PASS。

- [ ] **Step 7: 提交**

```bash
git add backend/product/handler/internal.go backend/product/main.go backend/product/handler/internal_test.go
git commit -m "feat(product): internal live-streams batch returns viewer_count"
```

---

## Task 2: auction-service client 透传 viewer_count

**Files:**
- Modify: `backend/auction/client/live_stream_client.go:16-22`
- Test: `backend/auction/client/live_stream_client_test.go`（若不存在则创建）

- [ ] **Step 1: 写失败测试**

先确认是否存在 `backend/auction/client/live_stream_client_test.go`；若无则创建，内容：

```go
package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPLiveStreamClient_BatchDecodesViewerCount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"message":"success","data":{"items":[{"id":101,"name":"room","cover_image":"c","status":1,"creator_id":9,"viewer_count":128}]}}`))
	}))
	defer srv.Close()

	c := NewHTTPLiveStreamClient(srv.URL, 0)
	out, err := c.BatchGetLiveStreams(context.Background(), []int64{101})
	require.NoError(t, err)
	require.Contains(t, out, int64(101))
	assert.Equal(t, int64(128), out[101].ViewerCount)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend/auction && go test ./client/ -run TestHTTPLiveStreamClient_BatchDecodesViewerCount -v`
Expected: 编译失败（`LiveStreamSummary` 无 `ViewerCount` 字段）。

- [ ] **Step 3: 加字段**

`backend/auction/client/live_stream_client.go`，`LiveStreamSummary` 改为：

```go
type LiveStreamSummary struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	CoverImage  string `json:"cover_image"`
	Status      int    `json:"status"`
	CreatorID   int64  `json:"creator_id"`
	ViewerCount int64  `json:"viewer_count"`
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd backend/auction && go test ./client/ -run TestHTTPLiveStreamClient_BatchDecodesViewerCount -v`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add backend/auction/client/live_stream_client.go backend/auction/client/live_stream_client_test.go
git commit -m "feat(auction): live stream client carries viewer_count"
```

---

## Task 3: auction-service 列表编排回填 viewer_count（含降级）

**Files:**
- Modify: `backend/auction/handler/auction_list.go`
- Modify: `backend/auction/handler/auction.go:449`
- Test: `backend/auction/handler/auction_list_test.go`

- [ ] **Step 1: 写失败测试**

在 `backend/auction/handler/auction_list_test.go` 顶部（其它 fake 之后）新增替身：

```go
// fakeLiveStreamClient 是 client.LiveStreamClient 的可控替身。
type fakeLiveStreamClient struct {
	out        map[int64]client.LiveStreamSummary
	err        error
	calledIDs  []int64
}

func (f *fakeLiveStreamClient) BatchGetLiveStreams(_ context.Context, ids []int64) (map[int64]client.LiveStreamSummary, error) {
	f.calledIDs = append([]int64(nil), ids...)
	if f.err != nil {
		return nil, f.err
	}
	return f.out, nil
}
```

并新增测试函数（独立于 `TestBuildAuctionListResponse`，使用新 6 参签名）：

```go
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
		require.NoError(t, err) // 关键：不报错
		assert.Equal(t, int64(1), total)
		require.Len(t, items, 1)
		assert.Equal(t, int64(0), items[0].ViewerCount)
		assert.Equal(t, "p1", items[0].Product.Name) // 商品摘要正常
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
		assert.Empty(t, lsc.calledIDs) // 无 live_stream_id → 不发起批量
	})
}
```

- [ ] **Step 2: 同步既有 7 处调用签名**

将 `auction_list_test.go` 中现有 7 处 `BuildAuctionListResponse(ctx, fp, fl.List, ...)`（行 118/157/176/191/204/216/236）改为在 `fp` 后插入 `nil`（即 lsc 占位），形如：

```go
BuildAuctionListResponse(ctx, fp, nil, fl.List, rf, params)
```

- [ ] **Step 3: 运行测试确认失败**

Run: `cd backend/auction && go test ./handler/ -run TestBuildAuctionListResponse -v`
Expected: 编译失败（`BuildAuctionListResponse` 仍是 5 参 / `AuctionListItem` 无 `ViewerCount`）。

- [ ] **Step 4: 改 auction_list.go——加字段 + 参数 + Step3.5 + 回填**

4a. import 增加 `"log"`（与现有 `"context" "fmt"` 同组）。

4b. `AuctionListItem` 加字段：

```go
type AuctionListItem struct {
	model.Auction
	StartPrice  *decimal.Decimal      `json:"start_price,omitempty"`
	Product     AuctionProductSummary `json:"product"`
	ViewerCount int64                 `json:"viewer_count"`
}
```

4c. 函数签名加 `lsc client.LiveStreamClient`（紧跟 `pc`）：

```go
func BuildAuctionListResponse(
	ctx context.Context,
	pc client.ProductClient,
	lsc client.LiveStreamClient,
	lister auctionLister,
	ruleFetcher auctionRuleBatchFetcher,
	p ListParams,
) ([]AuctionListItem, int64, error) {
```

4d. 在 Step 3（batch product summaries）之后、Step 4 回填之前插入：

```go
	// Step 3.5: 批量取直播间观看人数（仅 viewer_count）。
	// 降级语义：失败不阻断整页，缺省为 0（装饰性信息不让整页挂）。
	viewerByStream := map[int64]int64{}
	if lsc != nil {
		streamIDs := make([]int64, 0, len(auctions))
		seenStream := make(map[int64]struct{}, len(auctions))
		for _, a := range auctions {
			if a.LiveStreamID == nil || *a.LiveStreamID <= 0 {
				continue
			}
			if _, ok := seenStream[*a.LiveStreamID]; ok {
				continue
			}
			seenStream[*a.LiveStreamID] = struct{}{}
			streamIDs = append(streamIDs, *a.LiveStreamID)
		}
		if len(streamIDs) > 0 {
			if streams, lerr := lsc.BatchGetLiveStreams(ctx, streamIDs); lerr != nil {
				log.Printf("[WARN] auction list: batch live streams for viewer_count failed (degraded): %v", lerr)
			} else {
				for id, s := range streams {
					viewerByStream[id] = s.ViewerCount
				}
			}
		}
	}
```

4e. Step 4 回填循环里，在 `out = append(out, item)` 之前加（即 `if rule...` 块之后）：

```go
			if a.LiveStreamID != nil {
				item.ViewerCount = viewerByStream[*a.LiveStreamID]
			}
```

- [ ] **Step 5: 改 auction.go 调用处**

`backend/auction/handler/auction.go:449`，把：

```go
		items, total, err := BuildAuctionListResponse(ctx, h.productClient, h.auctionService.ListAuctionsWithFilters, h.ruleFetcher, params)
```

改为：

```go
		items, total, err := BuildAuctionListResponse(ctx, h.productClient, h.liveStreamClient, h.auctionService.ListAuctionsWithFilters, h.ruleFetcher, params)
```

> 生产环境 `h.liveStreamClient` 已在 `main.go:208` 通过 `SetLiveStreamClient` 注入，无需改 main。

- [ ] **Step 6: 运行测试确认通过**

Run: `cd backend/auction && go test ./handler/ -run TestBuildAuctionListResponse -v`
Expected: 全部 PASS（新用例 + 既有用例不回归）。

- [ ] **Step 7: 全量编译 + 包测试**

Run: `cd backend/auction && go build ./... && go test ./handler/ ./client/`
Expected: PASS。

- [ ] **Step 8: 提交**

```bash
git add backend/auction/handler/auction_list.go backend/auction/handler/auction.go backend/auction/handler/auction_list_test.go
git commit -m "feat(auction): backfill live viewer_count into auction list with graceful degradation"
```

---

## Task 4: H5 首页类型扩展 + 进行中卡片渲染观看人数 pill

**Files:**
- Modify: `frontend/h5/src/pages/Home/index.tsx`（接口 RawAuction/HomeAuction、normalizeAuction、普通卡片渲染）
- Modify: `frontend/h5/src/pages/Home/Home.module.css`
- Test: `frontend/h5/src/pages/Home/__tests__/Home.test.tsx`

- [ ] **Step 1: 写失败测试**

在 `Home.test.tsx` 的 `describe('HomePage 分类联动 (T2.10)', ...)` 内追加 3 个用例：

```go
  it('进行中竞拍卡片在封面右下角展示真实观看人数', async () => {
    mockedAuctionApi.list.mockResolvedValue({
      list: [
        {
          id: 30, product_id: 300, live_stream_id: 5, status: 1,
          current_price: 1000, viewer_count: 128,
          end_time: new Date(Date.now() + 3_600_000).toISOString(),
          product: { id: 300, name: '在线人数拍品', images: ['/a.jpg'] },
        },
      ],
      total: 1,
    });

    renderHome();

    await screen.findByRole('heading', { name: '在线人数拍品' });
    expect(screen.getByText(/128\s*观看/)).toBeInTheDocument();
  });

  it('进行中竞拍 viewer_count 为 0（降级）时不展示观看人数', async () => {
    mockedAuctionApi.list.mockResolvedValue({
      list: [
        {
          id: 31, product_id: 301, live_stream_id: 5, status: 1,
          current_price: 1000, viewer_count: 0,
          end_time: new Date(Date.now() + 3_600_000).toISOString(),
          product: { id: 301, name: '降级拍品', images: ['/a.jpg'] },
        },
      ],
      total: 1,
    });

    renderHome();

    await screen.findByRole('heading', { name: '降级拍品' });
    expect(screen.queryByText(/观看/)).not.toBeInTheDocument();
  });

  it('已结束竞拍即使带 viewer_count 也不展示观看人数', async () => {
    mockedAuctionApi.list.mockResolvedValue({
      list: [
        {
          id: 32, product_id: 302, status: 3,
          current_price: 1000, viewer_count: 99,
          end_time: new Date(Date.now() - 1000).toISOString(),
          product: { id: 302, name: '结束拍品', images: ['/a.jpg'] },
        },
      ],
      total: 1,
    });

    renderHome();

    await screen.findByRole('heading', { name: '结束拍品' });
    expect(screen.queryByText(/观看/)).not.toBeInTheDocument();
  });
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend/h5 && npx jest src/pages/Home/__tests__/Home.test.tsx -t '观看'`
Expected: FAIL（页面未渲染观看人数）。

- [ ] **Step 3: 扩展类型 + normalize**

`frontend/h5/src/pages/Home/index.tsx`：

3a. `RawAuction` 接口加字段（在 `bid_count?` 附近）：

```go
  viewer_count?: number | string;
```

3b. `HomeAuction` 接口加字段：

```go
  viewerCount: number;
```

3c. `normalizeAuction` 的返回对象加（在 `bidCount` 行附近）：

```go
    viewerCount: toNumber(auction.viewer_count),
```

- [ ] **Step 4: 渲染 pill**

`index.tsx` 普通卡片 `.imageWrapper` 内，`statusBadge` 那个 `</div>` 之后、`imageWrapper` 闭合 `</div>` 之前插入：

```go
                    {statusInfo.live && auction.viewerCount > 0 && (
                      <div className={styles.viewerBadge}>
                        <span className={styles.viewerDot} />
                        {auction.viewerCount.toLocaleString()} 观看
                      </div>
                    )}
```

- [ ] **Step 5: 加样式**

`frontend/h5/src/pages/Home/Home.module.css`，在 `.statusBadge {...}` 块之后追加：

```css
.viewerBadge {
  position: absolute;
  right: var(--spacing-2);
  bottom: var(--spacing-2);
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px var(--spacing-2);
  border-radius: var(--radius-full);
  background: rgba(0, 0, 0, 0.62);
  color: #fff;
  font-size: 10px;
  backdrop-filter: blur(12px);
}

.viewerDot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #ff4d4f;
}
```

- [ ] **Step 6: 运行测试确认通过**

Run: `cd frontend/h5 && npx jest src/pages/Home/__tests__/Home.test.tsx`
Expected: 全部 PASS（新用例 + 既有用例不回归）。

- [ ] **Step 7: 类型检查 + 构建**

Run: `cd frontend/h5 && npx tsc --noEmit && npm run build`
Expected: 无类型错误，构建成功。

- [ ] **Step 8: 提交**

```bash
git add frontend/h5/src/pages/Home/index.tsx frontend/h5/src/pages/Home/Home.module.css frontend/h5/src/pages/Home/__tests__/Home.test.tsx
git commit -m "feat(h5): show real viewer count on live auction cards"
```

---

## Task 5: 本地联调与部署验证

**Files:** 无代码改动（验证 only）。

- [ ] **Step 1: 本地部署**

Run: `bash scripts/deploy-dev.sh`（或项目既定的 `/dp-dev` 流程）
Expected: gateway / product / auction / h5 容器健康。

- [ ] **Step 2: 验证接口透出 viewer_count**

Run: `curl -s 'http://localhost:8080/api/v1/auctions?page=1&page_size=5' | python3 -m json.tool | grep -A1 viewer_count`
Expected: 进行中竞拍 item 含 `viewer_count` 字段（有真实直播间人数时 > 0）。

- [ ] **Step 3: 验证降级（可选）**

临时停 product-service 后请求 `/auctions`，预期仍 200、列表正常、`viewer_count` 为 0；恢复 product-service。

- [ ] **Step 4: H5 视觉验证**

浏览器打开 H5 首页，确认进行中卡片封面右下角出现「◉ N 观看」pill，待开始/已结束卡片无 pill。

---

## Self-Review

**Spec coverage：**
- §4.1 product 内部接口加 viewer_count → Task 1 ✅
- §4.2 auction client 透传 → Task 2 ✅
- §4.3 编排回填 + 降级（仅 viewer_count 降级、商品摘要仍 5xx、nil 跳过、去重）→ Task 3 ✅
- §4.4 前端类型 + 渲染（双条件）+ CSS → Task 4 ✅
- §5 契约变更 → Task 1/2/3 体现 ✅
- §6 TDD 三层测试点 → Task 1 Step1 / Task 3 Step1 / Task 4 Step1 ✅
- 边界（直播间维度多卡片同值 / 无需分批 / 日志降噪）→ Task 3 实现已遵守 ✅

**Placeholder scan：** 无 TBD/TODO；每个代码步骤均含完整代码与命令。

**Type consistency：** `ViewerCount int64`（Go 三处 JSON tag 统一 `viewer_count`）、`viewerCount: number`（前端）、`liveViewerCounter.ViewerCountForLiveStream` 与 `LiveStreamService` 既有方法签名一致、`BuildAuctionListResponse` 新 6 参签名在 Task 3 调用处与全部测试同步。

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-06-10-h5-home-auction-viewer-count.md`. Two execution options:

1. **Subagent-Driven (recommended)** — 每个 Task 派发独立 subagent，Task 间双阶段评审，快速迭代。
2. **Inline Execution** — 本会话内按 executing-plans 批量执行带 checkpoint 评审。

Which approach?
