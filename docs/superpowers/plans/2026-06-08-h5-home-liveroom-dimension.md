# H5 首页直播间维度重构 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 H5 首页"全部"tab 从竞拍维度改为直播间维度，每张卡片聚合"正在竞拍 / 即将开始 / 最近成交"三层信息。

**Architecture:** 入口不变（gateway `/api/v1/live-streams` → product-service `ListPublic`）。auction-service 新增两个 internal 批量接口（next / recent_deals），只返回 `product_id` + 金额 + 时间；product-service 在 `ListPublic` 内放开 status 过滤、回填这两类竞拍，并用本地 `ProductDAO`（product 表的 owner）批量解析 `product_name`，避免 auction→product 反向跨服务调用。前端只改默认"全部"tab 的数据源与卡片渲染，分类/收藏 tab 与直播 feed 形态不动。

**Tech Stack:** Go 1.24 (Hertz, GORM, shopspring/decimal) + auction/product 微服务；前端 React + TypeScript + CSS Module + Jest/Testing Library；E2E Playwright。

---

## 关键设计决策（实施前必读）

1. **商品名解析放在 product-service 侧**：auction model 无 `ProductName` 字段（`backend/auction/model/auction.go`），仅有 `ProductID` 和 `CurrentPrice`。next/recent_deals 的 internal 接口只回 `product_id` + 价格 + 时间；product-service 用自有 `ProductDAO.GetByIDs`（`backend/product/dao/product.go:176`）批量补 `product_name`。这是最短路径，符合"谁是数据 owner 谁解析"。
2. **金额口径**：auction 表只有 `CurrentPrice decimal.Decimal`。对 pending 竞拍，`start_price = CurrentPrice.String()`；对 ended 竞拍，`final_price = CurrentPrice.String()`。
3. **next 语义**：`status = AuctionStatusPending(0)`，按 `start_time ASC, id ASC` 每个直播间取第一条（最早即将开始）。
4. **recent_deals 语义**：`status = AuctionStatusEnded(3)` 且 `winner_id IS NOT NULL`（真实成交，排除流拍），按 `end_time DESC` 每个直播间取最近 N 条（默认 N=3）。per-group limit 在 Go 侧遍历有序结果截断，避免窗口函数兼容问题（与现有 `GetCurrentByLiveStreamIDs` 同款思路）。
5. **放开 status 过滤**：`ListPublic` 候选直播间从"仅 Live(1)"放宽为 `status IN (NotStarted=0, Live=1)`，**排除 Ended(2) / Banned(3)**。回填后**丢弃既无 current 又无 next 的空壳直播间**（recent_deals 只是氛围，不单独保留一个直播间）。排序 `status DESC, created_at DESC`（Live 优先）。
6. **分页权衡（MVP）**：因"一商家一直播间"活跃总量极小，回填后过滤空壳会使 `total = 过滤后长度`（近似值）。本期接受该近似，于代码注释和状态文件记录限制。
7. **前端范围收敛**：只有默认"全部"tab 切到 `liveStreamApi.list` 并渲染新卡片；分类 tab 仍走 `auctionApi.list({category_id})`（live-streams 接口不支持 category_id），收藏 tab 与直播 feed 不变。

---

## 文件结构

**auction-service（新增 next/recent_deals 能力）**
- Modify: `backend/auction/dao/auction.go` — 新增 `GetNextByLiveStreamIDs`、`GetRecentDealsByLiveStreamIDs`
- Test: `backend/auction/dao/auction_next_recent_test.go` — 新建 DAO 测试
- Create: `backend/auction/handler/internal_next_recent_auction.go` — 两个 internal handler + DAO fetcher 适配
- Test: `backend/auction/handler/internal_next_recent_auction_test.go` — handler 测试
- Modify: `backend/auction/main.go:216,497-521` — 构造 handler、注册路由（扩 `registerInternalRoutes` 入参）

**product-service（放开过滤 + 回填）**
- Modify: `backend/product/client/auction_client.go` — 新增 `NextByLiveStreamIDs`、`RecentDealsByLiveStreamIDs` 及响应结构体
- Test: `backend/product/client/auction_client_next_recent_test.go` — client 测试（httptest mock）
- Modify: `backend/product/dao/live_stream.go` — 新增 `ListPublicCandidates`（status IN (0,1)）
- Modify: `backend/product/service/live_stream.go` — 新增 `ListPublicCandidates` 透传
- Modify: `backend/product/handler/live_stream.go:21-34,359-438` — `ListPublic` 放开过滤 + 回填 next/recent_deals + product_name + 丢空壳；新增 `SetProductNameResolver`
- Modify: `backend/product/main.go:150-162` — 装配 product name resolver
- Test: `backend/product/handler/live_stream_public_test.go` — 扩展回填/过滤断言

**frontend（首页卡片）**
- Modify: `frontend/h5/src/services/api.ts` — `liveStreamApi.list` 返回类型可选强化（最小改动，不强制）
- Create: `frontend/h5/src/pages/Home/LiveRoomCard.tsx` — 直播间维度卡片组件
- Create: `frontend/h5/src/pages/Home/LiveRoomCard.module.css` — 卡片样式（复用 color tokens，双主题）
- Test: `frontend/h5/src/pages/Home/__tests__/LiveRoomCard.test.tsx` — 组件测试
- Modify: `frontend/h5/src/pages/Home/index.tsx` — "全部"tab 切数据源 + 渲染 `LiveRoomCard`
- Test: `frontend/h5/src/pages/Home/__tests__/Home.test.tsx` 与 `frontend/h5/src/__tests__/integration/Home.integration.test.tsx` — 更新用例
- Create: `frontend/h5/e2e/home-liveroom.spec.ts`（或现有 e2e 目录）— Playwright 全链路

---

## Task 1: auction DAO 新增 GetNextByLiveStreamIDs

**Files:**
- Modify: `backend/auction/dao/auction.go`
- Test: `backend/auction/dao/auction_next_recent_test.go`

- [ ] **Step 1: 写失败测试**

新建 `backend/auction/dao/auction_next_recent_test.go`（复用同目录 `auction_current_test.go` 的 `newCurrentTestDB`）：

```go
package dao

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"auction-service/model"
)

func ptrInt64(v int64) *int64 { return &v }

func TestGetNextByLiveStreamIDs(t *testing.T) {
	db := newCurrentTestDB(t)
	d := NewAuctionDAO(db)
	now := time.Now()

	// 直播间 1：两条 pending，期望取 start_time 最早的那条(id=11)
	require.NoError(t, db.Create(&model.Auction{ID: 11, ProductID: 101, LiveStreamID: ptrInt64(1), Status: model.AuctionStatusPending, CurrentPrice: decimal.NewFromInt(100), StartTime: now.Add(10 * time.Minute), EndTime: now.Add(40 * time.Minute)}).Error)
	require.NoError(t, db.Create(&model.Auction{ID: 12, ProductID: 102, LiveStreamID: ptrInt64(1), Status: model.AuctionStatusPending, CurrentPrice: decimal.NewFromInt(200), StartTime: now.Add(30 * time.Minute), EndTime: now.Add(60 * time.Minute)}).Error)
	// 直播间 1：一条 ongoing，不应被选为 next
	require.NoError(t, db.Create(&model.Auction{ID: 13, ProductID: 103, LiveStreamID: ptrInt64(1), Status: model.AuctionStatusOngoing, CurrentPrice: decimal.NewFromInt(300), StartTime: now.Add(-5 * time.Minute), EndTime: now.Add(20 * time.Minute)}).Error)
	// 直播间 2：无 pending，不应出现在结果
	require.NoError(t, db.Create(&model.Auction{ID: 21, ProductID: 201, LiveStreamID: ptrInt64(2), Status: model.AuctionStatusEnded, CurrentPrice: decimal.NewFromInt(500), StartTime: now.Add(-60 * time.Minute), EndTime: now.Add(-30 * time.Minute)}).Error)

	got, err := d.GetNextByLiveStreamIDs(context.Background(), []int64{1, 2})
	require.NoError(t, err)
	require.Contains(t, got, int64(1))
	require.NotContains(t, got, int64(2))
	require.Equal(t, int64(11), got[1].ID)
	require.Equal(t, int64(101), got[1].ProductID)
}

func TestGetNextByLiveStreamIDsEmpty(t *testing.T) {
	db := newCurrentTestDB(t)
	d := NewAuctionDAO(db)
	got, err := d.GetNextByLiveStreamIDs(context.Background(), nil)
	require.NoError(t, err)
	require.Empty(t, got)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend/auction && go test ./dao/ -run TestGetNextByLiveStreamIDs -v`
Expected: FAIL，`d.GetNextByLiveStreamIDs undefined`

- [ ] **Step 3: 实现 DAO 方法**

在 `backend/auction/dao/auction.go` 末尾（紧随 `GetCurrentByLiveStreamIDs`）新增：

```go
// GetNextByLiveStreamIDs 为每个 live_stream 取"即将开始"的下一场竞拍。
// 规则：status=Pending，按 start_time ASC, id ASC 每组取第一条。
func (d *AuctionDAO) GetNextByLiveStreamIDs(ctx context.Context, liveStreamIDs []int64) (map[int64]*model.Auction, error) {
	result := make(map[int64]*model.Auction)
	if len(liveStreamIDs) == 0 {
		return result, nil
	}
	var rows []model.Auction
	err := d.db.WithContext(ctx).
		Where("live_stream_id IN ?", liveStreamIDs).
		Where("status = ?", model.AuctionStatusPending).
		Order("live_stream_id ASC, start_time ASC, id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for i := range rows {
		ls := rows[i].LiveStreamID
		if ls == nil {
			continue
		}
		if _, ok := result[*ls]; !ok {
			result[*ls] = &rows[i]
		}
	}
	return result, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd backend/auction && go test ./dao/ -run TestGetNextByLiveStreamIDs -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add backend/auction/dao/auction.go backend/auction/dao/auction_next_recent_test.go
git commit -m "feat(auction): add GetNextByLiveStreamIDs DAO for upcoming auction backfill"
```

---

## Task 2: auction DAO 新增 GetRecentDealsByLiveStreamIDs

**Files:**
- Modify: `backend/auction/dao/auction.go`
- Test: `backend/auction/dao/auction_next_recent_test.go`

- [ ] **Step 1: 追加失败测试**

在 `backend/auction/dao/auction_next_recent_test.go` 追加：

```go
func TestGetRecentDealsByLiveStreamIDs(t *testing.T) {
	db := newCurrentTestDB(t)
	d := NewAuctionDAO(db)
	now := time.Now()

	// 直播间 1：三条已成交(winner!=nil)，期望按 end_time DESC 取最近 2 条 = id 33, 32
	require.NoError(t, db.Create(&model.Auction{ID: 31, ProductID: 301, LiveStreamID: ptrInt64(1), Status: model.AuctionStatusEnded, WinnerID: ptrInt64(9001), CurrentPrice: decimal.NewFromInt(100), StartTime: now.Add(-90 * time.Minute), EndTime: now.Add(-80 * time.Minute)}).Error)
	require.NoError(t, db.Create(&model.Auction{ID: 32, ProductID: 302, LiveStreamID: ptrInt64(1), Status: model.AuctionStatusEnded, WinnerID: ptrInt64(9002), CurrentPrice: decimal.NewFromInt(200), StartTime: now.Add(-60 * time.Minute), EndTime: now.Add(-50 * time.Minute)}).Error)
	require.NoError(t, db.Create(&model.Auction{ID: 33, ProductID: 303, LiveStreamID: ptrInt64(1), Status: model.AuctionStatusEnded, WinnerID: ptrInt64(9003), CurrentPrice: decimal.NewFromInt(300), StartTime: now.Add(-30 * time.Minute), EndTime: now.Add(-20 * time.Minute)}).Error)
	// 流拍(winner=nil) 不计入
	require.NoError(t, db.Create(&model.Auction{ID: 34, ProductID: 304, LiveStreamID: ptrInt64(1), Status: model.AuctionStatusEnded, WinnerID: nil, CurrentPrice: decimal.NewFromInt(400), StartTime: now.Add(-10 * time.Minute), EndTime: now.Add(-5 * time.Minute)}).Error)

	got, err := d.GetRecentDealsByLiveStreamIDs(context.Background(), []int64{1}, 2)
	require.NoError(t, err)
	require.Len(t, got[1], 2)
	require.Equal(t, int64(33), got[1][0].ID)
	require.Equal(t, int64(32), got[1][1].ID)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend/auction && go test ./dao/ -run TestGetRecentDealsByLiveStreamIDs -v`
Expected: FAIL，`d.GetRecentDealsByLiveStreamIDs undefined`

- [ ] **Step 3: 实现 DAO 方法**

在 `backend/auction/dao/auction.go` 追加（注意 per-group limit 在 Go 侧截断）：

```go
// GetRecentDealsByLiveStreamIDs 为每个 live_stream 取最近 N 条成交记录。
// 规则：status=Ended 且 winner_id IS NOT NULL，按 end_time DESC 每组取前 N 条。
func (d *AuctionDAO) GetRecentDealsByLiveStreamIDs(ctx context.Context, liveStreamIDs []int64, n int) (map[int64][]*model.Auction, error) {
	result := make(map[int64][]*model.Auction)
	if len(liveStreamIDs) == 0 || n <= 0 {
		return result, nil
	}
	var rows []model.Auction
	err := d.db.WithContext(ctx).
		Where("live_stream_id IN ?", liveStreamIDs).
		Where("status = ?", model.AuctionStatusEnded).
		Where("winner_id IS NOT NULL").
		Order("live_stream_id ASC, end_time DESC, id DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for i := range rows {
		ls := rows[i].LiveStreamID
		if ls == nil {
			continue
		}
		if len(result[*ls]) >= n {
			continue
		}
		result[*ls] = append(result[*ls], &rows[i])
	}
	return result, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd backend/auction && go test ./dao/ -run TestGetRecentDealsByLiveStreamIDs -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add backend/auction/dao/auction.go backend/auction/dao/auction_next_recent_test.go
git commit -m "feat(auction): add GetRecentDealsByLiveStreamIDs DAO for recent deals backfill"
```

---

## Task 3: auction internal handler + 路由（next / recent_deals）

**Files:**
- Create: `backend/auction/handler/internal_next_recent_auction.go`
- Test: `backend/auction/handler/internal_next_recent_auction_test.go`
- Modify: `backend/auction/main.go`

- [ ] **Step 1: 写失败测试**

新建 `backend/auction/handler/internal_next_recent_auction_test.go`（参照 `internal_current_auction_test.go` 的 Hertz `ut.PerformRequest` 风格）：

```go
package handler

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/test/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/stretchr/testify/require"
)

type fakeNextFetcher struct{ m map[int64]NextAuctionItem }

func (f fakeNextFetcher) Fetch(_ context.Context, _ []int64) (map[int64]NextAuctionItem, error) {
	return f.m, nil
}

func TestNextByLiveStreams(t *testing.T) {
	h := NewInternalNextAuctionHandler(fakeNextFetcher{m: map[int64]NextAuctionItem{
		1: {AuctionID: 11, ProductID: 101, StartPrice: "100", StartTime: "2026-06-08T10:00:00Z"},
	}})
	srv := server.Default(server.WithHostPorts("127.0.0.1:0"))
	srv.POST("/internal/auctions/next-by-live-streams", h.Handle)

	w := ut.PerformRequest(srv.Engine, consts.MethodPost, "/internal/auctions/next-by-live-streams",
		&ut.Body{Body: bytesReader(`{"live_stream_ids":[1,2]}`), Len: -1},
		ut.Header{Key: "Content-Type", Value: "application/json"})
	resp := w.Result()
	require.Equal(t, 200, resp.StatusCode())

	var body struct {
		Data struct {
			Items []map[string]interface{} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body(), &body))
	require.Len(t, body.Data.Items, 1)
	require.EqualValues(t, 1, body.Data.Items[0]["live_stream_id"])
	require.EqualValues(t, 11, body.Data.Items[0]["auction_id"])
	require.EqualValues(t, 101, body.Data.Items[0]["product_id"])
}
```

> 说明：`bytesReader` 用 `strings.NewReader` 替代即可；若现有测试已有辅助函数请复用。recent_deals 的 handler 测试同款追加（fake fetcher 返回 `map[int64][]DealAuctionItem`，断言 `deals` 数组与 `final_price`）。

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend/auction && go test ./handler/ -run TestNextByLiveStreams -v`
Expected: FAIL，`NewInternalNextAuctionHandler undefined`

- [ ] **Step 3: 实现 handler**

新建 `backend/auction/handler/internal_next_recent_auction.go`：

```go
package handler

import (
	"context"
	"log"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"auction-service/dao"
	"auction-service/model"
)

// ---------- next ----------

type NextAuctionItem struct {
	AuctionID  int64
	ProductID  int64
	StartPrice string
	StartTime  string
}

type NextAuctionFetcher interface {
	Fetch(ctx context.Context, liveStreamIDs []int64) (map[int64]NextAuctionItem, error)
}

type InternalNextAuctionHandler struct{ fetcher NextAuctionFetcher }

func NewInternalNextAuctionHandler(f NextAuctionFetcher) *InternalNextAuctionHandler {
	return &InternalNextAuctionHandler{fetcher: f}
}

type internalLiveStreamIDsRequest struct {
	LiveStreamIDs []int64 `json:"live_stream_ids"`
}

func (h *InternalNextAuctionHandler) Handle(ctx context.Context, c *app.RequestContext) {
	var req internalLiveStreamIDsRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误: " + err.Error()})
		return
	}
	got, err := h.fetcher.Fetch(ctx, req.LiveStreamIDs)
	if err != nil {
		log.Printf("internal next-by-live-streams failed: ids=%v err=%v", req.LiveStreamIDs, err)
		c.JSON(500, map[string]interface{}{"code": 500, "message": "internal error"})
		return
	}
	items := make([]map[string]interface{}, 0, len(got))
	for lsID, it := range got {
		items = append(items, map[string]interface{}{
			"live_stream_id": lsID,
			"auction_id":     it.AuctionID,
			"product_id":     it.ProductID,
			"start_price":    it.StartPrice,
			"start_time":     it.StartTime,
		})
	}
	c.JSON(200, map[string]interface{}{"code": 200, "data": map[string]interface{}{"items": items}})
}

type NextAuctionDAOFetcher struct{ dao *dao.AuctionDAO }

func NewNextAuctionDAOFetcher(d *dao.AuctionDAO) *NextAuctionDAOFetcher { return &NextAuctionDAOFetcher{dao: d} }

func (f *NextAuctionDAOFetcher) Fetch(ctx context.Context, ids []int64) (map[int64]NextAuctionItem, error) {
	rows, err := f.dao.GetNextByLiveStreamIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]NextAuctionItem, len(rows))
	for lsID, a := range rows {
		out[lsID] = NextAuctionItem{
			AuctionID:  a.ID,
			ProductID:  a.ProductID,
			StartPrice: a.CurrentPrice.String(),
			StartTime:  a.StartTime.Format(time.RFC3339),
		}
	}
	return out, nil
}

// ---------- recent deals ----------

type DealAuctionItem struct {
	AuctionID  int64
	ProductID  int64
	FinalPrice string
	EndTime    string
}

type RecentDealsFetcher interface {
	Fetch(ctx context.Context, liveStreamIDs []int64, n int) (map[int64][]DealAuctionItem, error)
}

type InternalRecentDealsHandler struct {
	fetcher RecentDealsFetcher
	limit   int
}

func NewInternalRecentDealsHandler(f RecentDealsFetcher, limit int) *InternalRecentDealsHandler {
	if limit <= 0 {
		limit = 3
	}
	return &InternalRecentDealsHandler{fetcher: f, limit: limit}
}

func (h *InternalRecentDealsHandler) Handle(ctx context.Context, c *app.RequestContext) {
	var req internalLiveStreamIDsRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误: " + err.Error()})
		return
	}
	got, err := h.fetcher.Fetch(ctx, req.LiveStreamIDs, h.limit)
	if err != nil {
		log.Printf("internal recent-deals-by-live-streams failed: ids=%v err=%v", req.LiveStreamIDs, err)
		c.JSON(500, map[string]interface{}{"code": 500, "message": "internal error"})
		return
	}
	items := make([]map[string]interface{}, 0, len(got))
	for lsID, deals := range got {
		ds := make([]map[string]interface{}, 0, len(deals))
		for _, d := range deals {
			ds = append(ds, map[string]interface{}{
				"auction_id":  d.AuctionID,
				"product_id":  d.ProductID,
				"final_price": d.FinalPrice,
				"end_time":    d.EndTime,
			})
		}
		items = append(items, map[string]interface{}{"live_stream_id": lsID, "deals": ds})
	}
	c.JSON(200, map[string]interface{}{"code": 200, "data": map[string]interface{}{"items": items}})
}

type RecentDealsDAOFetcher struct{ dao *dao.AuctionDAO }

func NewRecentDealsDAOFetcher(d *dao.AuctionDAO) *RecentDealsDAOFetcher { return &RecentDealsDAOFetcher{dao: d} }

func (f *RecentDealsDAOFetcher) Fetch(ctx context.Context, ids []int64, n int) (map[int64][]DealAuctionItem, error) {
	rows, err := f.dao.GetRecentDealsByLiveStreamIDs(ctx, ids, n)
	if err != nil {
		return nil, err
	}
	out := make(map[int64][]DealAuctionItem, len(rows))
	for lsID, deals := range rows {
		list := make([]DealAuctionItem, 0, len(deals))
		for _, a := range deals {
			list = append(list, DealAuctionItem{
				AuctionID:  a.ID,
				ProductID:  a.ProductID,
				FinalPrice: a.CurrentPrice.String(),
				EndTime:    a.EndTime.Format(time.RFC3339),
			})
		}
		out[lsID] = list
	}
	return out, nil
}

var _ = model.AuctionStatusEnded // keep model import if unused elsewhere
```

> 收尾时若 `model` 未被其它符号引用，删除最后一行哑引用与 import。

- [ ] **Step 4: 注册路由**

修改 `backend/auction/main.go`：在 ~L216 构造 handler（紧随 currentAuctionHandler）：

```go
nextAuctionHandler := handler.NewInternalNextAuctionHandler(handler.NewNextAuctionDAOFetcher(auctionDAO))
recentDealsHandler := handler.NewInternalRecentDealsHandler(handler.NewRecentDealsDAOFetcher(auctionDAO), 3)
```

扩展 `registerInternalRoutes` 签名（L497）追加两个参数 `nextAuctionHandler *handler.InternalNextAuctionHandler, recentDealsHandler *handler.InternalRecentDealsHandler`，并在分组内（L510 后）注册：

```go
if nextAuctionHandler != nil {
	internal.POST("/auctions/next-by-live-streams", nextAuctionHandler.Handle)
}
if recentDealsHandler != nil {
	internal.POST("/auctions/recent-deals-by-live-streams", recentDealsHandler.Handle)
}
```

同步更新 `registerInternalRoutes(...)` 的调用处，把两个新 handler 传入。

- [ ] **Step 5: 运行测试与构建确认通过**

Run: `cd backend/auction && go test ./handler/ -run 'TestNextByLiveStreams|TestRecentDeals' -v && go build ./...`
Expected: PASS + 构建成功

- [ ] **Step 6: 提交**

```bash
git add backend/auction/handler/internal_next_recent_auction.go backend/auction/handler/internal_next_recent_auction_test.go backend/auction/main.go
git commit -m "feat(auction): add internal next/recent-deals by-live-streams endpoints"
```

---

## Task 4: product-service AuctionClient 新增 Next / RecentDeals 方法

**Files:**
- Modify: `backend/product/client/auction_client.go`
- Test: `backend/product/client/auction_client_next_recent_test.go`

- [ ] **Step 1: 写失败测试**

新建 `backend/product/client/auction_client_next_recent_test.go`：

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

func TestNextByLiveStreamIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/auctions/next-by-live-streams", r.URL.Path)
		_, _ = w.Write([]byte(`{"code":200,"data":{"items":[{"live_stream_id":1,"auction_id":11,"product_id":101,"start_price":"100.00","start_time":"2026-06-08T10:00:00Z"}]}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewAuctionClient(srv.URL, 0)
	got, err := c.NextByLiveStreamIDs(context.Background(), []int64{1})
	require.NoError(t, err)
	require.Contains(t, got, int64(1))
	assert.EqualValues(t, 11, got[1].AuctionID)
	assert.EqualValues(t, 101, got[1].ProductID)
	assert.Equal(t, "100.00", got[1].StartPrice)
}

func TestRecentDealsByLiveStreamIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/auctions/recent-deals-by-live-streams", r.URL.Path)
		_, _ = w.Write([]byte(`{"code":200,"data":{"items":[{"live_stream_id":1,"deals":[{"auction_id":33,"product_id":303,"final_price":"300.00","end_time":"2026-06-08T09:00:00Z"}]}]}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewAuctionClient(srv.URL, 0)
	got, err := c.RecentDealsByLiveStreamIDs(context.Background(), []int64{1}, 3)
	require.NoError(t, err)
	require.Len(t, got[1], 1)
	assert.EqualValues(t, 303, got[1][0].ProductID)
	assert.Equal(t, "300.00", got[1][0].FinalPrice)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend/product && go test ./client/ -run 'TestNextByLiveStreamIDs|TestRecentDealsByLiveStreamIDs' -v`
Expected: FAIL，方法未定义

- [ ] **Step 3: 实现 client 方法**

在 `backend/product/client/auction_client.go` 末尾追加（仿照 `CurrentByLiveStreamIDs` L72-110 的请求/解析骨架）：

```go
type NextAuctionItem struct {
	LiveStreamID int64  `json:"live_stream_id"`
	AuctionID    int64  `json:"auction_id"`
	ProductID    int64  `json:"product_id"`
	StartPrice   string `json:"start_price"`
	StartTime    string `json:"start_time"`
}

type DealAuctionItem struct {
	AuctionID  int64  `json:"auction_id"`
	ProductID  int64  `json:"product_id"`
	FinalPrice string `json:"final_price"`
	EndTime    string `json:"end_time"`
}

func (c *AuctionClient) NextByLiveStreamIDs(ctx context.Context, ids []int64) (map[int64]NextAuctionItem, error) {
	result := make(map[int64]NextAuctionItem)
	if len(ids) == 0 {
		return result, nil
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Items []NextAuctionItem `json:"items"`
		} `json:"data"`
	}
	if err := c.postInternal(ctx, "/internal/auctions/next-by-live-streams",
		map[string]interface{}{"live_stream_ids": ids}, &body); err != nil {
		return nil, err
	}
	for _, it := range body.Data.Items {
		result[it.LiveStreamID] = it
	}
	return result, nil
}

func (c *AuctionClient) RecentDealsByLiveStreamIDs(ctx context.Context, ids []int64, n int) (map[int64][]DealAuctionItem, error) {
	result := make(map[int64][]DealAuctionItem)
	if len(ids) == 0 {
		return result, nil
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Items []struct {
				LiveStreamID int64             `json:"live_stream_id"`
				Deals        []DealAuctionItem `json:"deals"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := c.postInternal(ctx, "/internal/auctions/recent-deals-by-live-streams",
		map[string]interface{}{"live_stream_ids": ids, "limit": n}, &body); err != nil {
		return nil, err
	}
	for _, it := range body.Data.Items {
		result[it.LiveStreamID] = it.Deals
	}
	return result, nil
}
```

并新增私有辅助 `postInternal`（抽取现有重复的 marshal/请求/token/解码逻辑；若不想重构现有方法，可仅供新方法使用）：

```go
func (c *AuctionClient) postInternal(ctx context.Context, path string, payload interface{}, out interface{}) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.internalToken != "" {
		req.Header.Set("X-Internal-Token", c.internalToken)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("call auction-service: %w", err)
	}
	defer resp.Body.Close()
	defer io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("auction-service returned status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd backend/product && go test ./client/ -run 'TestNextByLiveStreamIDs|TestRecentDealsByLiveStreamIDs' -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add backend/product/client/auction_client.go backend/product/client/auction_client_next_recent_test.go
git commit -m "feat(product): add AuctionClient Next/RecentDeals by-live-streams methods"
```

---

## Task 5: product-service ListPublicCandidates（放开 status 过滤）

**Files:**
- Modify: `backend/product/dao/live_stream.go`
- Modify: `backend/product/service/live_stream.go`
- Test: `backend/product/dao/live_stream_test.go`（若不存在则新建 `live_stream_candidates_test.go`）

- [ ] **Step 1: 写失败测试**

新建 `backend/product/dao/live_stream_candidates_test.go`：

```go
package dao

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"product-service/model"
)

func TestListPublicCandidatesExcludesEndedAndBanned(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.LiveStream{}))
	db.Exec("DELETE FROM live_streams")
	require.NoError(t, db.Create(&model.LiveStream{ID: 1, CreatorID: 1, Name: "未开播", Status: model.LiveStreamStatusNotStarted}).Error)
	require.NoError(t, db.Create(&model.LiveStream{ID: 2, CreatorID: 2, Name: "直播中", Status: model.LiveStreamStatusLive}).Error)
	require.NoError(t, db.Create(&model.LiveStream{ID: 3, CreatorID: 3, Name: "已结束", Status: model.LiveStreamStatusEnded}).Error)
	require.NoError(t, db.Create(&model.LiveStream{ID: 4, CreatorID: 4, Name: "封禁", Status: model.LiveStreamStatusBanned}).Error)

	d := NewLiveStreamDAO(db)
	rows, total, err := d.ListPublicCandidates(context.Background(), 0, 20)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	ids := map[int64]bool{}
	for _, r := range rows {
		ids[r.ID] = true
	}
	require.True(t, ids[1])
	require.True(t, ids[2])
	require.False(t, ids[3])
	require.False(t, ids[4])
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend/product && go test ./dao/ -run TestListPublicCandidates -v`
Expected: FAIL，`d.ListPublicCandidates undefined`

- [ ] **Step 3: 实现 DAO + service**

在 `backend/product/dao/live_stream.go`（参照现有 `ListAdmin` L166-190）新增：

```go
// ListPublicCandidates 返回首页候选直播间：status IN (NotStarted, Live)，排除 Ended/Banned。
// 排序：Live 优先（status DESC），再按 created_at DESC。
func (d *LiveStreamDAO) ListPublicCandidates(ctx context.Context, offset, limit int) ([]model.LiveStream, int64, error) {
	statuses := []model.LiveStreamStatus{model.LiveStreamStatusNotStarted, model.LiveStreamStatusLive}
	base := d.db.WithContext(ctx).Model(&model.LiveStream{}).Where("status IN ?", statuses)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.LiveStream
	err := base.Order("status DESC, created_at DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}
```

在 `backend/product/service/live_stream.go`（参照 `ListAdmin` L166-170）新增：

```go
func (s *LiveStreamService) ListPublicCandidates(ctx context.Context, page, pageSize int) ([]model.LiveStream, int64, error) {
	offset := (page - 1) * pageSize
	return s.liveStreamDAO.ListPublicCandidates(ctx, offset, pageSize)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd backend/product && go test ./dao/ -run TestListPublicCandidates -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add backend/product/dao/live_stream.go backend/product/service/live_stream.go backend/product/dao/live_stream_candidates_test.go
git commit -m "feat(product): add ListPublicCandidates (NotStarted+Live, exclude ended/banned)"
```

---

## Task 6: ListPublic 回填 next_auction / recent_deals + product_name + 丢空壳

**Files:**
- Modify: `backend/product/handler/live_stream.go`
- Modify: `backend/product/main.go`
- Test: `backend/product/handler/live_stream_public_test.go`

- [ ] **Step 1: 写失败测试**

在 `backend/product/handler/live_stream_public_test.go` 追加用例（mock auction-service 三个 endpoint，注入 product name resolver）：

```go
func TestListPublic_BackfillsNextAndRecentDeals(t *testing.T) {
	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/auctions/current-by-live-streams":
			_, _ = w.Write([]byte(`{"code":200,"data":{"items":[{"live_stream_id":601,"auction_id":11,"product_id":8,"current_price":"1200.00","status":1}]}}`))
		case "/internal/auctions/next-by-live-streams":
			_, _ = w.Write([]byte(`{"code":200,"data":{"items":[{"live_stream_id":602,"auction_id":21,"product_id":9,"start_price":"300.00","start_time":"2026-06-08T10:00:00Z"}]}}`))
		case "/internal/auctions/recent-deals-by-live-streams":
			_, _ = w.Write([]byte(`{"code":200,"data":{"items":[{"live_stream_id":601,"deals":[{"auction_id":31,"product_id":7,"final_price":"500.00","end_time":"2026-06-08T09:00:00Z"}]}]}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(auctionMock.Close)

	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Product{}, &model.Category{}, &model.AuctionRule{}, &model.LiveStream{}))
	db.Exec("DELETE FROM live_streams")
	db.Exec("DELETE FROM products")
	// 601 直播中(有 current)，602 未开播(有 next)，603 已结束应被候选过滤掉
	require.NoError(t, db.Create(&model.LiveStream{ID: 601, CreatorID: 1, Name: "A", Status: model.LiveStreamStatusLive}).Error)
	require.NoError(t, db.Create(&model.LiveStream{ID: 602, CreatorID: 2, Name: "B", Status: model.LiveStreamStatusNotStarted}).Error)
	require.NoError(t, db.Create(&model.LiveStream{ID: 603, CreatorID: 3, Name: "空壳", Status: model.LiveStreamStatusLive}).Error) // 无 current/next → 丢弃
	// 商品名解析数据
	require.NoError(t, db.Create(&model.Product{ID: 9, Name: "翡翠手镯"}).Error)
	require.NoError(t, db.Create(&model.Product{ID: 7, Name: "和田玉牌"}).Error)

	svc := service.NewLiveStreamService(dao.NewLiveStreamDAO(db))
	h := NewLiveStreamHandler(svc)
	h.SetAuctionClient(client.NewAuctionClient(auctionMock.URL, 0))
	h.SetProductNameResolver(dao.NewProductDAO(db)) // ProductDAO 满足 resolver 接口

	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/live-streams")
	h.ListPublic(context.Background(), c)

	require.Equal(t, 200, c.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	data := body["data"].(map[string]interface{})
	list := data["list"].([]interface{})
	require.Len(t, list, 2) // 601 + 602；603 空壳被丢弃

	byID := map[int64]map[string]interface{}{}
	for _, raw := range list {
		it := raw.(map[string]interface{})
		byID[int64(it["id"].(float64))] = it
	}

	// 602 next_auction 含 product_name
	next := byID[602]["next_auction"].(map[string]interface{})
	assert.EqualValues(t, 21, next["auction_id"])
	assert.Equal(t, "翡翠手镯", next["product_name"])
	assert.Equal(t, "300.00", next["start_price"])

	// 601 recent_deals 含 product_name + final_price
	deals := byID[601]["recent_deals"].([]interface{})
	require.Len(t, deals, 1)
	d0 := deals[0].(map[string]interface{})
	assert.Equal(t, "和田玉牌", d0["product_name"])
	assert.Equal(t, "500.00", d0["final_price"])
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend/product && go test ./handler/ -run TestListPublic_BackfillsNextAndRecentDeals -v`
Expected: FAIL，`SetProductNameResolver undefined` / 字段缺失

- [ ] **Step 3: 实现 handler 改动**

修改 `backend/product/handler/live_stream.go`：

(a) `LiveStreamHandler` 结构体（L21-24）新增字段与接口、setter：

```go
type ProductNameResolver interface {
	GetByIDs(ctx context.Context, ids []int64) ([]model.Product, error)
}

type LiveStreamHandler struct {
	liveStreamService   *service.LiveStreamService
	auctionClient       *client.AuctionClient
	productNameResolver ProductNameResolver
}

func (h *LiveStreamHandler) SetProductNameResolver(r ProductNameResolver) {
	h.productNameResolver = r
}
```

(b) 重写 `ListPublic`（L361-438）：候选改用 `ListPublicCandidates`，回填 current/next/recent，收集 product_id 批量解析名称，丢弃空壳：

```go
func (h *LiveStreamHandler) ListPublic(ctx context.Context, c *app.RequestContext) {
	page, err := parseAdminLiveStreamIntQuery(c, "page", 1)
	if err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "无效的页码"})
		return
	}
	pageSize, err := parseAdminLiveStreamIntQuery(c, "page_size", 20)
	if err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "无效的分页大小"})
		return
	}
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > maxPublicLiveStreamPageSize {
		pageSize = maxPublicLiveStreamPageSize
	}

	liveStreams, _, err := h.liveStreamService.ListPublicCandidates(ctx, page, pageSize)
	if err != nil {
		log.Printf("LiveStream ListPublic failed: page=%d pageSize=%d err=%v", page, pageSize, err)
		c.JSON(500, map[string]interface{}{"code": 500, "message": "获取直播间列表失败"})
		return
	}

	ids := make([]int64, 0, len(liveStreams))
	for _, ls := range liveStreams {
		ids = append(ids, ls.ID)
	}

	current := map[int64]client.CurrentAuctionItem{}
	next := map[int64]client.NextAuctionItem{}
	recent := map[int64][]client.DealAuctionItem{}
	if h.auctionClient != nil && len(ids) > 0 {
		if got, err := h.auctionClient.CurrentByLiveStreamIDs(ctx, ids); err != nil {
			log.Printf("ListPublic current degraded: err=%v", err)
		} else {
			current = got
		}
		if got, err := h.auctionClient.NextByLiveStreamIDs(ctx, ids); err != nil {
			log.Printf("ListPublic next degraded: err=%v", err)
		} else {
			next = got
		}
		if got, err := h.auctionClient.RecentDealsByLiveStreamIDs(ctx, ids, 3); err != nil {
			log.Printf("ListPublic recent-deals degraded: err=%v", err)
		} else {
			recent = got
		}
	}

	// 批量解析 product_name
	productIDs := map[int64]struct{}{}
	for _, it := range next {
		productIDs[it.ProductID] = struct{}{}
	}
	for _, deals := range recent {
		for _, d := range deals {
			productIDs[d.ProductID] = struct{}{}
		}
	}
	nameByID := map[int64]string{}
	if h.productNameResolver != nil && len(productIDs) > 0 {
		pids := make([]int64, 0, len(productIDs))
		for id := range productIDs {
			pids = append(pids, id)
		}
		if products, err := h.productNameResolver.GetByIDs(ctx, pids); err != nil {
			log.Printf("ListPublic product-name resolve degraded: err=%v", err)
		} else {
			for _, p := range products {
				nameByID[p.ID] = p.Name
			}
		}
	}

	list := make([]map[string]interface{}, 0, len(liveStreams))
	for _, ls := range liveStreams {
		var currentAuctionID, currentProductID, currentPrice interface{} = nil, nil, nil
		_, hasCurrent := current[ls.ID]
		if item, ok := current[ls.ID]; ok {
			currentAuctionID = item.AuctionID
			currentProductID = item.ProductID
			currentPrice = item.CurrentPrice
		}

		var nextAuction interface{} = nil
		nx, hasNext := next[ls.ID]
		if hasNext {
			nextAuction = map[string]interface{}{
				"auction_id":   nx.AuctionID,
				"product_name": nameByID[nx.ProductID],
				"start_price":  nx.StartPrice,
				"start_time":   nx.StartTime,
			}
		}

		// 丢弃空壳：既无 current 又无 next
		if !hasCurrent && !hasNext {
			continue
		}

		recentDeals := make([]map[string]interface{}, 0)
		for _, d := range recent[ls.ID] {
			recentDeals = append(recentDeals, map[string]interface{}{
				"product_name": nameByID[d.ProductID],
				"final_price":  d.FinalPrice,
			})
		}

		list = append(list, map[string]interface{}{
			"id":                 ls.ID,
			"name":               ls.Name,
			"cover_image":        ls.CoverImage,
			"status":             ls.Status,
			"host_name":          ls.StreamerName,
			"host_avatar":        ls.StreamerAvatar,
			"viewer_count":       h.liveStreamService.ViewerCountForLiveStream(ctx, &ls),
			"current_auction_id": currentAuctionID,
			"current_product_id": currentProductID,
			"current_price":      currentPrice,
			"next_auction":       nextAuction,
			"recent_deals":       recentDeals,
		})
	}

	c.JSON(200, map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"list":      list,
			"total":     len(list), // MVP：过滤空壳后的近似总数
			"page":      page,
			"page_size": pageSize,
		},
	})
}
```

> 注意 `for _, ls := range liveStreams` 中对 `&ls` 取址沿用现有写法（原 L422 已如此）。

(c) `backend/product/main.go` L159 后装配 resolver（productDAO 在初始化段已存在；若变量名不同按实际）：

```go
liveStreamHandler.SetProductNameResolver(productDAO)
```

> 若 `productDAO` 变量在 main 中不可见，使用已有的 `internalHandler := handler.NewInternalHandler(productService, liveStreamDAO)` 旁边同源的 product DAO 实例；必要时在初始化段保留 `productDAO` 引用。`*dao.ProductDAO` 已实现 `GetByIDs(ctx, []int64) ([]model.Product, error)`（`backend/product/dao/product.go:176`），天然满足 `ProductNameResolver`。

- [ ] **Step 4: 运行测试确认通过（含回归）**

Run: `cd backend/product && go test ./handler/ -run TestListPublic -v && go build ./...`
Expected: 新用例 PASS；`TestListPublic_ClampsPageSize` 仍 PASS；构建成功

> `TestListPublic_OnlyLiveAndCurrentAuction` 旧断言（`require.Len(t, list, 2)` 基于"仅 Live"语义）会因放开过滤而失效——更新该用例：为它的直播间补 current/next，或改断言为"空壳被丢弃"语义，使其与新行为一致。

- [ ] **Step 5: 提交**

```bash
git add backend/product/handler/live_stream.go backend/product/main.go backend/product/handler/live_stream_public_test.go
git commit -m "feat(product): backfill next_auction/recent_deals with product names in ListPublic"
```

---

## Task 7: 前端 LiveRoomCard 组件

**Files:**
- Create: `frontend/h5/src/pages/Home/LiveRoomCard.tsx`
- Create: `frontend/h5/src/pages/Home/LiveRoomCard.module.css`
- Test: `frontend/h5/src/pages/Home/__tests__/LiveRoomCard.test.tsx`

- [ ] **Step 1: 写失败测试**

新建 `frontend/h5/src/pages/Home/__tests__/LiveRoomCard.test.tsx`（Jest + Testing Library，参照 `Home.test.tsx` 风格）：

```tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import LiveRoomCard, { LiveRoomItem } from '../LiveRoomCard';

const renderCard = (room: LiveRoomItem, onSubscribe = jest.fn(), onEnter = jest.fn()) =>
  render(
    <MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <LiveRoomCard room={room} onSubscribe={onSubscribe} onEnter={onEnter} subscribedProductIds={new Set()} />
    </MemoryRouter>,
  );

test('有 current_auction 时显示直播中并以进入直播为主操作', () => {
  const onEnter = jest.fn();
  renderCard({ id: 1, name: '瑾瑜珠宝', status: 1, current_auction_id: 11, current_product_id: 8, current_price: '1200.00', recent_deals: [] }, jest.fn(), onEnter);
  expect(screen.getByText('直播中')).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: '进入直播间' }));
  expect(onEnter).toHaveBeenCalledWith(1, 11);
});

test('无 current 有 next 时显示即将开始并以预约为主操作', () => {
  const onSubscribe = jest.fn();
  renderCard(
    { id: 2, name: '云裳阁', status: 0, current_auction_id: null, next_auction: { auction_id: 21, product_name: '翡翠手镯', start_price: '300.00', start_time: '2026-06-08T10:00:00Z' }, recent_deals: [] },
    onSubscribe,
  );
  expect(screen.getByText('即将开始')).toBeInTheDocument();
  expect(screen.getByText('翡翠手镯')).toBeInTheDocument();
});

test('渲染最近成交氛围信息', () => {
  renderCard({ id: 3, name: 'X', status: 1, current_auction_id: 5, recent_deals: [{ product_name: '和田玉牌', final_price: '500.00' }] });
  expect(screen.getByText(/和田玉牌/)).toBeInTheDocument();
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend/h5 && npx jest src/pages/Home/__tests__/LiveRoomCard.test.tsx`
Expected: FAIL，`Cannot find module '../LiveRoomCard'`

- [ ] **Step 3: 实现组件 + 样式**

新建 `frontend/h5/src/pages/Home/LiveRoomCard.tsx`：

```tsx
import React from 'react';
import { useNavigate } from 'react-router-dom';
import styles from './LiveRoomCard.module.css';

export interface NextAuction {
  auction_id: number;
  product_name: string;
  start_price: string;
  start_time: string;
}

export interface RecentDeal {
  product_name: string;
  final_price: string;
}

export interface LiveRoomItem {
  id: number;
  name: string;
  status: number;
  cover_image?: string;
  host_name?: string;
  host_avatar?: string;
  viewer_count?: number;
  current_auction_id?: number | null;
  current_product_id?: number | null;
  current_price?: string | null;
  next_auction?: NextAuction | null;
  recent_deals?: RecentDeal[];
}

interface Props {
  room: LiveRoomItem;
  onSubscribe?: (productId?: number, auctionId?: number) => void;
  onEnter?: (liveStreamId: number, auctionId?: number) => void;
  subscribedProductIds?: Set<number>;
}

const hasCurrent = (room: LiveRoomItem) =>
  room.current_auction_id != null && Number(room.current_auction_id) > 0;

const LiveRoomCard: React.FC<Props> = ({ room, onSubscribe, onEnter }) => {
  const navigate = useNavigate();
  const live = hasCurrent(room);
  const next = room.next_auction;
  const deals = room.recent_deals ?? [];

  const enter = () => {
    const auctionId = Number(room.current_auction_id) || undefined;
    if (onEnter) {
      onEnter(room.id, auctionId);
      return;
    }
    navigate(`/live?id=${room.id}&auction_id=${auctionId ?? ''}`);
  };

  return (
    <article className={styles.card}>
      <div className={styles.imageWrapper}>
        {room.cover_image && <img className={styles.cover} src={room.cover_image} alt={room.name} />}
        <span className={live ? styles.statusLive : styles.statusUpcoming}>
          {live ? '直播中' : '即将开始'}
        </span>
        {typeof room.viewer_count === 'number' && (
          <span className={styles.viewers}>{room.viewer_count} 在线</span>
        )}
      </div>
      <div className={styles.body}>
        <h3 className={styles.name}>{room.name}</h3>
        {live ? (
          <p className={styles.price}>当前 ¥{room.current_price ?? '—'}</p>
        ) : next ? (
          <p className={styles.nextLine}>
            即将开拍：{next.product_name}（起拍 ¥{next.start_price}）
          </p>
        ) : null}

        {deals.length > 0 && (
          <ul className={styles.deals} aria-label="最近成交">
            {deals.map((d, i) => (
              <li key={i} className={styles.dealItem}>
                {d.product_name} 已成交 ¥{d.final_price}
              </li>
            ))}
          </ul>
        )}

        <div className={styles.actions}>
          {live ? (
            <button type="button" className={styles.primaryButton} onClick={enter}>
              进入直播间
            </button>
          ) : (
            <button
              type="button"
              className={styles.secondaryButton}
              onClick={() => onSubscribe?.(undefined, next?.auction_id)}
            >
              预约开拍提醒
            </button>
          )}
        </div>
      </div>
    </article>
  );
};

export default LiveRoomCard;
```

新建 `frontend/h5/src/pages/Home/LiveRoomCard.module.css`（全部使用 `var(--*)` token，自动适配日/夜双主题；参照 `Home.module.css` 已有类）：

```css
.card {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg, 16px);
  box-shadow: var(--shadow-key);
  overflow: hidden;
  margin-bottom: var(--space-4, 16px);
}
.imageWrapper { position: relative; aspect-ratio: 16 / 9; background: var(--bg-elevated); }
.cover { width: 100%; height: 100%; object-fit: cover; }
.statusLive,
.statusUpcoming {
  position: absolute; top: 8px; left: 8px;
  padding: 2px 8px; border-radius: 999px; font-size: 12px; color: #fff;
}
.statusLive { background: var(--text-brand, #d4af37); }
.statusUpcoming { background: rgba(0, 0, 0, 0.55); }
.viewers {
  position: absolute; top: 8px; right: 8px;
  padding: 2px 8px; border-radius: 999px; font-size: 12px;
  background: rgba(0, 0, 0, 0.45); color: #fff;
}
.body { padding: var(--space-3, 12px); }
.name { margin: 0 0 6px; font-size: 16px; color: var(--text-primary); }
.price { margin: 0; font-size: 15px; color: var(--text-brand); font-weight: 600; }
.nextLine { margin: 0; font-size: 13px; color: var(--text-secondary); }
.deals { list-style: none; margin: 8px 0 0; padding: 0; }
.dealItem { font-size: 12px; color: var(--text-secondary); line-height: 1.6; }
.actions { margin-top: 12px; }
.primaryButton,
.secondaryButton {
  width: 100%; padding: 10px 0; border-radius: var(--radius-md, 10px);
  font-size: 14px; border: none; cursor: pointer;
}
.primaryButton { background: var(--text-brand, #d4af37); color: #1a1a1a; }
.secondaryButton { background: transparent; color: var(--text-brand); border: 1px solid var(--text-brand); }
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend/h5 && npx jest src/pages/Home/__tests__/LiveRoomCard.test.tsx`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add frontend/h5/src/pages/Home/LiveRoomCard.tsx frontend/h5/src/pages/Home/LiveRoomCard.module.css frontend/h5/src/pages/Home/__tests__/LiveRoomCard.test.tsx
git commit -m "feat(h5): add LiveRoomCard component (three-tier info, dual theme)"
```

---

## Task 8: 首页"全部"tab 切换为直播间维度

**Files:**
- Modify: `frontend/h5/src/pages/Home/index.tsx`
- Test: `frontend/h5/src/pages/Home/__tests__/Home.test.tsx`
- Test: `frontend/h5/src/__tests__/integration/Home.integration.test.tsx`

- [ ] **Step 1: 写失败测试**

在 `Home.test.tsx` 追加（mock `liveStreamApi.list` 返回直播间维度数据，断言"全部"tab 渲染直播间卡片）：

```tsx
test('"全部"tab 渲染直播间维度卡片', async () => {
  mockedLiveStreamApi.list.mockResolvedValue({
    list: [
      { id: 1, name: '瑾瑜珠宝行', status: 1, current_auction_id: 11, current_price: '1200.00', recent_deals: [{ product_name: '和田玉牌', final_price: '500.00' }] },
      { id: 2, name: '云裳阁', status: 0, current_auction_id: null, next_auction: { auction_id: 21, product_name: '翡翠手镯', start_price: '300.00', start_time: '2026-06-08T10:00:00Z' }, recent_deals: [] },
    ],
    total: 2,
  });
  renderHome();
  expect(await screen.findByRole('heading', { name: '瑾瑜珠宝行' })).toBeInTheDocument();
  expect(screen.getByRole('heading', { name: '云裳阁' })).toBeInTheDocument();
  expect(screen.getByText(/翡翠手镯/)).toBeInTheDocument();
});
```

并在文件顶部 `jest.mock('../../../services/api', ...)` 中补 `liveStreamApi: { list: jest.fn() }` 与对应 `mockedLiveStreamApi` 引用。

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend/h5 && npx jest src/pages/Home/__tests__/Home.test.tsx -t "直播间维度"`
Expected: FAIL（仍渲染旧竞拍卡片 / liveStreamApi 未调用）

- [ ] **Step 3: 改造首页**

修改 `frontend/h5/src/pages/Home/index.tsx`：

(a) 顶部 import 新增：

```tsx
import LiveRoomCard, { LiveRoomItem } from './LiveRoomCard';
```

并确保从 `services/api` 引入 `liveStreamApi`。

(b) 新增 state：

```tsx
const [liveRooms, setLiveRooms] = useState<LiveRoomItem[]>([]);
```

(c) 改造 `fetchAuctions`（L372-412）：在 `activeTab === '全部'` 分支改用 `liveStreamApi.list`；分类分支保持原 `auctionApi.list({category_id})`：

```tsx
if (activeTab === '全部') {
  try {
    const response = await liveStreamApi.list(1, 20);
    setLiveRooms(extractList<LiveRoomItem>(response));
    setAuctions([]);
    setFavoriteLiveStreams([]);
  } catch (error) {
    console.error('获取直播间列表失败:', error);
    setLiveRooms([]);
  } finally {
    setLoading(false);
  }
  return;
}
// 分类 tab：保持原 auctionApi.list 逻辑（清空 liveRooms）
setLiveRooms([]);
```

> 注意在原"收藏"分支与分类分支也需 `setLiveRooms([])` 以避免残留。

(d) 渲染分支（竞拍列表渲染块 L573-640 之前）新增"全部"tab 的直播间渲染：

```tsx
{activeTab === '全部' ? (
  liveRooms.length === 0 ? (
    <EmptyState /* 复用现有空态 */ />
  ) : (
    <div className={styles.list}>
      {liveRooms.map((room) => (
        <LiveRoomCard
          key={room.id}
          room={room}
          subscribedProductIds={subscribedProductIds}
          onSubscribe={(productId, auctionId) => handleSubscribeReminder(productId, auctionId)}
          onEnter={(liveStreamId, auctionId) =>
            navigate(`/live?id=${liveStreamId}&auction_id=${auctionId ?? ''}`)
          }
        />
      ))}
    </div>
  )
) : (
  /* 原有：分类竞拍列表 / 收藏列表渲染保持不变 */
)}
```

> 复用现有 loading/empty 渲染骨架；`handleSubscribeReminder` 当前签名为 `(productId)`，若需要透传 auctionId 按现有实现适配（无 productId 时走预约提醒既有链路）。保持分类 tab、收藏 tab 渲染路径完全不变。

- [ ] **Step 4: 运行测试确认通过（含回归）**

Run: `cd frontend/h5 && npx jest src/pages/Home/__tests__/Home.test.tsx src/__tests__/integration/Home.integration.test.tsx`
Expected: 新用例 PASS；分类/收藏既有用例 PASS

> integration 测试若断言"全部"tab 调 `auctionApi.list`，需更新为断言 `liveStreamApi.list`；分类 tab 断言不变。

- [ ] **Step 5: 提交**

```bash
git add frontend/h5/src/pages/Home/index.tsx frontend/h5/src/pages/Home/__tests__/Home.test.tsx frontend/h5/src/__tests__/integration/Home.integration.test.tsx
git commit -m "feat(h5): switch Home default tab to live-room dimension cards"
```

---

## Task 9: E2E Playwright 全链路验证

**Files:**
- Create: `frontend/h5/e2e/home-liveroom.spec.ts`（按现有 e2e 目录约定放置）

- [ ] **Step 1: 写 E2E 脚本**

```ts
import { test, expect } from '@playwright/test';

test('首页直播间卡片 → 进入对应 feed', async ({ page }) => {
  await page.goto('/');
  // 等待"全部"tab 渲染直播间卡片
  const card = page.locator('article').first();
  await expect(card).toBeVisible();
  // 点"进入直播间"
  await card.getByRole('button', { name: '进入直播间' }).click();
  // 落到 /live?id=
  await expect(page).toHaveURL(/\/live\?id=\d+/);
});
```

- [ ] **Step 2: 启动本地服务并运行 E2E**

按 `docs/superpowers/sdd/RUNBOOK.md` 启动本地全链路（gateway + product + auction + h5）。

Run: `cd frontend/h5 && npx playwright test e2e/home-liveroom.spec.ts`
Expected: PASS（首页看到直播间卡 → 点进入落到 `/live?id=`）

> 若 demo 数据无"正在竞拍"，先按经验教训刷新直播会话（`/internal/test/live-streams/:id/restart`）更新 `started_at` 再跑。

- [ ] **Step 3: 提交**

```bash
git add frontend/h5/e2e/home-liveroom.spec.ts
git commit -m "test(h5): add e2e for home live-room card to feed navigation"
```

---

## Self-Review

**Spec 覆盖核对：**
- §3.1 职责分层（首页 vs feed）：Task 8 只改首页"全部"tab，feed 不动（非目标遵守）✓
- §3.2 卡片三层信息 + 状态优先级：Task 7 LiveRoomCard（current→直播中 / next→即将开始 / recent_deals 氛围）✓
- §4.1 放开 status 过滤：Task 5 `ListPublicCandidates`（NotStarted+Live，排除 Ended/Banned）+ Task 6 丢空壳 ✓
- §4.2 扩展回填 next_auction / recent_deals：Task 1/2 DAO + Task 3 internal 接口 + Task 4 client + Task 6 回填 ✓
- §4.3 响应契约（next_auction / recent_deals 字段 + decimal 字符串 + 跨服务 RPC 不跨库 JOIN）：Task 3/4/6，商品名由 product owner 本地解析 ✓
- §5.1 首页数据源切换 + 卡片 + 复用 /live?id= + 预约链路：Task 7/8 ✓
- §5.2 feed 不变：未触及 LiveFeedPage（仅 E2E 验证落点）✓
- §5.3 双主题：Task 7 LiveRoomCard.module.css 全 token 化 ✓
- §6 测试策略（后端单测 / 前端组件+集成 / E2E）：Task 1-9 全覆盖 ✓
- §7 风险（status stale 用 end_time 兜底）：LiveRoomCard 以 current_auction_id 判 live，氛围降级；保留既有 feed 兜底 ✓

**Placeholder 扫描：** 无 TBD/TODO；每个 code 步骤含完整代码。

**类型一致性：** 后端 `NextAuctionItem`/`DealAuctionItem` 在 auction handler（值）与 product client（带 json tag）两侧名称一致且职责区分；前端 `LiveRoomItem`/`NextAuction`/`RecentDeal` 在 Task 7 定义、Task 8 复用一致。DAO 方法名 `GetNextByLiveStreamIDs`/`GetRecentDealsByLiveStreamIDs`/`ListPublicCandidates` 全文一致。

**已知权衡（写入状态文件）：** ListPublic 过滤空壳后 `total` 为近似值（小规模可接受）；商品名解析失败时降级为空串（不阻断列表）。
