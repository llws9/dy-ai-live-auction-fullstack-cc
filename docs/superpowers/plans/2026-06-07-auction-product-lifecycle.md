# Auction Product Lifecycle Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将商品「发布」改为「可排期」，并让创建竞拍成为唯一创建 auction 记录、绑定直播间、校验商品归属与活跃唯一性的入口。

**Architecture:** 商品经营状态继续归 `product-service` 管理，竞拍交易过程继续归 `auction-service` 管理；两个服务只通过 `/internal/*` API 通信，不跨库直查。创建竞拍时由 `auction-service` 先调用 `product-service` 获取商品/规则事实和 active 直播间，再在本地事务写入 `auctions`；并发唯一性由应用层快速失败 + MySQL 生成列唯一索引兜底。

**Tech Stack:** Go 1.24+, Hertz, GORM, MySQL, sqlite test DB, React, TypeScript, Vitest, Testing Library.

---

## File Structure

- Modify: `backend/product/service/product.go`
  - 清理 `PublishProduct` 的直播间创建、active 校验、30 分钟默认开拍时间和规则读取副作用。
  - 新增 `GetProductAuctionInfo`，给 `auction-service` 查询商品归属、状态和已绑定规则。
- Modify: `backend/product/handler/product.go`
  - 发布接口继续兼容旧响应结构，但 `live_stream` 返回 `null`，成功文案改为「商品已进入竞拍池，可创建竞拍场次」。
- Modify: `backend/product/handler/internal.go`
  - 新增 `GET /internal/products/:id/auction-info`。
  - 新增 `POST /internal/live-streams/get-or-create`，按 `creator_id` 获取或创建 active 直播间。
- Modify: `backend/product/main.go`
  - 注册新增内部接口。
- Test: `backend/product/service/product_test.go`
  - 覆盖发布不创建直播间、草稿转可排期、非草稿失败。
- Test: `backend/product/handler/internal_test.go`
  - 覆盖商品 auction-info 和直播间 get-or-create 内部接口。

- Modify: `backend/auction/client/product_client.go`
  - 扩展 `ProductClient`，新增 `GetAuctionProductInfo` 与 `GetOrCreateActiveLiveStream`。
- Test: `backend/auction/client/product_client_test.go`
  - 用 `httptest.Server` 验证请求路径、Header、错误码、响应解码。

- Modify: `backend/auction/dao/auction.go`
  - 新增 `GetActiveByProductID`、`GetLatestTerminalByProductID`，支持创建前置校验。
  - 新增聚合筛选字段，支持 `竞拍中/已拍卖/流拍/已取消`。
- Create: `backend/auction/dao/auction_schema.go`
  - 增加 MySQL-only 的 `EnsureAuctionActiveProductUniqueIndex`，创建生成列 `active_product_key` 和唯一索引 `uk_active_product`；sqlite 单测跳过 DDL。
- Modify: `backend/auction/main.go`
  - `AutoMigrate` 后调用 `EnsureAuctionActiveProductUniqueIndex`。
- Test: `backend/auction/dao/auction_test.go`
  - 覆盖 active/latest terminal 查询。
- Test: `backend/auction/dao/auction_schema_test.go`
  - 覆盖 sqlite 跳过、MySQL DDL 字符串和重复执行保护。

- Modify: `backend/auction/service/auction.go`
  - 扩展 `CreateAuctionRequest`，只接收 `ProductID`、`CreatorID`、`Duration`、`ProductOwnerID`、`ProductStatus`、`RuleBound`、`LiveStreamID`。
  - 实现 Fail-closed 校验：商家身份、归属、Published、规则已绑定、无活跃竞拍、最近成交不可再拍、直播间 ID 必须有效。
  - 插入 auction 时写 `LiveStreamID`，`StartTime=now`，`EndTime=now+duration`。
- Test: `backend/auction/service/auction_create_test.go`
  - 覆盖 §7.1 创建竞拍后端单测要求中的应用层规则。

- Modify: `backend/auction/handler/auction.go`
  - `CreateAuctionRequest` 移除 `start_price`、`increment` 业务使用，只接受 `product_id`、`duration`。
  - 校验 `X-User-ID` 和 `X-User-Role=merchant`。
  - 调 `productClient.GetAuctionProductInfo` 与 `productClient.GetOrCreateActiveLiveStream`。
  - 业务错误返回 `400/403/409`，内部错误返回 `500`。
- Test: `backend/auction/handler/auction_create_test.go`
  - 覆盖非商家失败、商品归属异常、直播间禁用、成功写入 live_stream_id。

- Create: `backend/auction/handler/internal_product_auctions.go`
  - 新增 `POST /internal/auctions/by-products`，返回每个 product 的 active auction 与 latest terminal auction，用于 product-service 派生商品展示状态。
- Modify: `backend/auction/main.go`
  - 注册 `/internal/auctions/by-products`。
- Test: `backend/auction/handler/internal_product_auctions_test.go`
  - 覆盖 active 优先、Ended+winner_id sold、Ended+nil unsold、Cancelled。

- Modify: `backend/product/client/auction_client.go`
  - 新增 `BatchProductAuctionStates(ctx, productIDs []int64)` 调 auction-service 内部接口。
- Modify: `backend/product/handler/product.go`
  - `AdminList` 将商品列表包装为响应 DTO，追加 `display_status`、`display_status_label`、`active_auction_id`、`latest_auction_id`、`latest_auction_result`。
- Modify: `backend/product/main.go`
  - 将 `AuctionClient` 注入 `ProductHandler`。
- Test: `backend/product/client/auction_client_test.go`
  - 覆盖 by-products 解码。
- Test: `backend/product/handler/product_test.go`
  - 覆盖商品派生状态优先级。

- Modify: `frontend/admin/src/shared/api/types.ts`
  - `Product` 增加派生状态字段，`Auction` 确认 `winner_id` 可选。
- Modify: `frontend/admin/src/pages-new/GoodsList.tsx`
  - 文案改为「设为可排期」。
  - 状态 Badge 使用后端 `display_status_label`。
  - 行操作按可排期、竞拍中、流拍、已拍卖调整。
- Modify: `frontend/admin/src/pages-new/AuctionList.tsx`
  - 筛选改为「全部场次/竞拍中/已拍卖/流拍/已取消」。
  - 创建表单说明改为 Spec 文案；无商品展示「暂无可排期商品」。
  - Badge 用 `status + winner_id` 映射已拍卖/流拍。
- Test: `frontend/admin/src/pages-new/__tests__/GoodsList.test.tsx`
  - 覆盖设为可排期文案与派生状态显示。
- Test: `frontend/admin/src/pages-new/__tests__/AuctionList.createAuction.test.tsx`
  - 覆盖无可选商品空态、错误中文展示、创建请求不带 `template_id/live_stream_id/start_time`。

- Modify: `backend/test/scenario/antisnipe/factory.go`
  - 增加注释固化「每个活跃竞拍 fixture 必须使用独立商品」。
- Create: `scripts/check-auction-product-consistency.sql`
  - 只读 SQL，检查 auction.product_id 缺失、creator_id 与 owner_id 不一致、一品多活跃竞拍。

---

### Task 1: Product Publish 只做 Draft → Published

**Files:**
- Modify: `backend/product/service/product.go:3-108`
- Modify: `backend/product/handler/product.go:394-453`
- Test: `backend/product/service/product_test.go`

- [ ] **Step 1: 写失败测试：发布商品不创建直播间、不校验 active 直播间**

在 `backend/product/service/product_test.go` 增加：

```go
func TestProductService_PublishProductOnlyMarksProductPublished(t *testing.T) {
	ctx := context.Background()
	db := newProductServiceTestDB(t)
	productDAO := dao.NewProductDAO(db)
	ruleDAO := dao.NewAuctionRuleDAO(db)
	liveStreamDAO := dao.NewLiveStreamDAO(db)
	svc := service.NewProductService(productDAO, ruleDAO, liveStreamDAO)

	product := &model.Product{
		OwnerID: ptrInt64(1001),
		Name:    "draft product",
		Status:  model.ProductStatusDraft,
	}
	require.NoError(t, db.Create(product).Error)

	got, liveStream, err := svc.PublishProduct(ctx, product.ID, 1001, nil)

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, model.ProductStatusPublished, got.Status)
	assert.Nil(t, liveStream)

	var count int64
	require.NoError(t, db.Model(&model.LiveStream{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestProductService_PublishProductRejectsNonDraft(t *testing.T) {
	ctx := context.Background()
	db := newProductServiceTestDB(t)
	svc := service.NewProductService(dao.NewProductDAO(db), dao.NewAuctionRuleDAO(db), dao.NewLiveStreamDAO(db))
	product := &model.Product{
		OwnerID: ptrInt64(1001),
		Name:    "published product",
		Status:  model.ProductStatusPublished,
	}
	require.NoError(t, db.Create(product).Error)

	got, liveStream, err := svc.PublishProduct(ctx, product.ID, 1001, nil)

	require.Error(t, err)
	assert.Nil(t, got)
	assert.Nil(t, liveStream)
	assert.Contains(t, err.Error(), "只有草稿状态的商品可以发布")
}
```

如果文件里没有测试 helper，补充：

```go
func ptrInt64(v int64) *int64 {
	return &v
}

func newProductServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(
		&model.Product{},
		&model.LiveStream{},
		&model.AuctionRule{},
	))
	return db
}
```

- [ ] **Step 2: 运行测试确认失败**

Run:

```bash
cd backend/product && go test ./service -run 'TestProductService_PublishProduct' -count=1
```

Expected: FAIL，当前实现会创建 `live_streams`，且 `liveStream != nil`。

- [ ] **Step 3: 最小实现：删除 PublishProduct 旧副作用**

将 `backend/product/service/product.go` 的 import 改为：

```go
import (
	"context"
	"errors"

	"product-service/dao"
	"product-service/model"
)
```

将 `PublishProduct` 整个函数替换为：

```go
// PublishProduct 将草稿商品设为可排期。
func (s *ProductService) PublishProduct(ctx context.Context, productID, creatorID int64, startTime *time.Time) (*model.Product, *model.LiveStream, error) {
	product, err := s.productDAO.GetByID(ctx, productID)
	if err != nil {
		return nil, nil, err
	}
	if product.Status != model.ProductStatusDraft {
		return nil, nil, errors.New("商品状态不正确，只有草稿状态的商品可以发布")
	}
	if product.OwnerID != nil && *product.OwnerID != creatorID {
		return nil, nil, errors.New("商品不存在或不属于当前商家")
	}

	product.Status = model.ProductStatusPublished
	if err := s.productDAO.Update(ctx, product); err != nil {
		return nil, nil, err
	}
	return product, nil, nil
}
```

保留 `startTime *time.Time` 参数是为了兼容现有 handler 签名；函数内部不再使用它。

将 `backend/product/handler/product.go` 的成功响应替换为：

```go
c.JSON(200, map[string]interface{}{
	"code":    200,
	"message": "商品已进入竞拍池，可创建竞拍场次",
	"data": map[string]interface{}{
		"product":     product,
		"live_stream": nil,
	},
})
```

- [ ] **Step 4: 运行测试确认通过**

Run:

```bash
cd backend/product && go test ./service -run 'TestProductService_PublishProduct' -count=1
```

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add backend/product/service/product.go backend/product/handler/product.go backend/product/service/product_test.go
git commit -m "fix(product): make publish only mark products schedulable"
```

---

### Task 2: Product Internal API 提供商品竞拍事实与 active 直播间

**Files:**
- Modify: `backend/product/service/product.go`
- Modify: `backend/product/handler/internal.go`
- Modify: `backend/product/main.go`
- Test: `backend/product/handler/internal_test.go`

- [ ] **Step 1: 写失败测试：auction-info 与 get-or-create active live stream**

在 `backend/product/handler/internal_test.go` 增加：

```go
func TestInternalHandler_GetAuctionProductInfo(t *testing.T) {
	ctx := context.Background()
	db := newInternalHandlerTestDB(t)
	productDAO := dao.NewProductDAO(db)
	ruleDAO := dao.NewAuctionRuleDAO(db)
	liveStreamDAO := dao.NewLiveStreamDAO(db)
	svc := service.NewProductService(productDAO, ruleDAO, liveStreamDAO)
	h := handler.NewInternalHandler(svc, liveStreamDAO)

	ownerID := int64(1001)
	product := &model.Product{OwnerID: &ownerID, Name: "schedulable", Status: model.ProductStatusPublished}
	require.NoError(t, db.Create(product).Error)
	require.NoError(t, db.Create(&model.AuctionRule{
		ProductID: product.ID,
		StartPrice: 100,
		Increment:  10,
		Duration:   3600,
	}).Error)

	app := server.Default()
	app.GET("/internal/products/:id/auction-info", h.GetAuctionProductInfo)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/internal/products/%d/auction-info", product.ID), nil)
	resp := performRequest(app, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body struct {
		Code int `json:"code"`
		Data struct {
			ID        int64 `json:"id"`
			OwnerID   int64 `json:"owner_id"`
			Status    int   `json:"status"`
			RuleBound bool  `json:"rule_bound"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, 200, body.Code)
	assert.Equal(t, product.ID, body.Data.ID)
	assert.Equal(t, ownerID, body.Data.OwnerID)
	assert.Equal(t, int(model.ProductStatusPublished), body.Data.Status)
	assert.True(t, body.Data.RuleBound)
}

func TestInternalHandler_GetOrCreateActiveLiveStream(t *testing.T) {
	db := newInternalHandlerTestDB(t)
	productDAO := dao.NewProductDAO(db)
	ruleDAO := dao.NewAuctionRuleDAO(db)
	liveStreamDAO := dao.NewLiveStreamDAO(db)
	svc := service.NewProductService(productDAO, ruleDAO, liveStreamDAO)
	h := handler.NewInternalHandler(svc, liveStreamDAO)

	app := server.Default()
	app.POST("/internal/live-streams/get-or-create", h.GetOrCreateActiveLiveStream)
	payload := strings.NewReader(`{"creator_id":1001,"creator_name":"merchant_1001"}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/live-streams/get-or-create", payload)
	req.Header.Set("Content-Type", "application/json")
	resp := performRequest(app, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body struct {
		Code int `json:"code"`
		Data struct {
			ID        int64 `json:"id"`
			CreatorID int64 `json:"creator_id"`
			Status    int   `json:"status"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, int64(1001), body.Data.CreatorID)
	assert.Equal(t, int(model.LiveStreamStatusActive), body.Data.Status)
	assert.NotZero(t, body.Data.ID)
}

func TestInternalHandler_GetOrCreateActiveLiveStreamRejectsBanned(t *testing.T) {
	db := newInternalHandlerTestDB(t)
	productDAO := dao.NewProductDAO(db)
	ruleDAO := dao.NewAuctionRuleDAO(db)
	liveStreamDAO := dao.NewLiveStreamDAO(db)
	svc := service.NewProductService(productDAO, ruleDAO, liveStreamDAO)
	h := handler.NewInternalHandler(svc, liveStreamDAO)

	// 预置一个已被禁用的直播间，GetOrCreate 应复用它并暴露 409，验证 §4.2「直播间 active」Fail-closed。
	require.NoError(t, db.Create(&model.LiveStream{
		CreatorID: 1001,
		Name:      "banned",
		Status:    model.LiveStreamStatusBanned,
	}).Error)

	app := server.Default()
	app.POST("/internal/live-streams/get-or-create", h.GetOrCreateActiveLiveStream)
	payload := strings.NewReader(`{"creator_id":1001,"creator_name":"merchant_1001"}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/live-streams/get-or-create", payload)
	req.Header.Set("Content-Type", "application/json")
	resp := performRequest(app, req)

	assert.Equal(t, http.StatusConflict, resp.Code)
}
```

> 注意：此用例依赖 `GetOrCreateByCreatorID` 在已存在直播间时返回 existing 记录（即使其被禁用）。执行 Step 前先读 `backend/product/dao/live_stream.go` 的 `GetOrCreateByCreatorID` 确认该行为；若它会把被禁用直播间「复活」为 Live，则需改为返回 existing 原状态，否则 409 分支不可达。

- [ ] **Step 2: 运行测试确认失败**

Run:

```bash
cd backend/product && go test ./handler -run 'TestInternalHandler_(GetAuctionProductInfo|GetOrCreateActiveLiveStream)' -count=1
```

Expected: FAIL，handler 方法未定义。

- [ ] **Step 3: 实现 service DTO 与查询**

在 `backend/product/service/product.go` 增加：

```go
type ProductAuctionInfo struct {
	ID        int64               `json:"id"`
	OwnerID   int64               `json:"owner_id"`
	Status    model.ProductStatus `json:"status"`
	RuleBound bool                `json:"rule_bound"`
}

func (s *ProductService) GetProductAuctionInfo(ctx context.Context, productID int64) (*ProductAuctionInfo, error) {
	product, err := s.productDAO.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if product.OwnerID == nil {
		return nil, errors.New("商品缺少 owner_id")
	}
	_, err = s.ruleDAO.GetByProductID(ctx, productID)
	ruleBound := err == nil
	return &ProductAuctionInfo{
		ID:        product.ID,
		OwnerID:   *product.OwnerID,
		Status:    product.Status,
		RuleBound: ruleBound,
	}, nil
}
```

- [ ] **Step 4: 实现 internal handler**

在 `backend/product/handler/internal.go` 增加：

```go
type getOrCreateLiveStreamRequest struct {
	CreatorID   int64  `json:"creator_id"`
	CreatorName string `json:"creator_name"`
}

func (h *InternalHandler) GetAuctionProductInfo(ctx context.Context, c *app.RequestContext) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "无效的商品ID"})
		return
	}
	info, err := h.productService.GetProductAuctionInfo(ctx, id)
	if err != nil {
		c.JSON(404, map[string]interface{}{"code": 404, "message": err.Error()})
		return
	}
	c.JSON(200, map[string]interface{}{"code": 200, "message": "success", "data": info})
}

func (h *InternalHandler) GetOrCreateActiveLiveStream(ctx context.Context, c *app.RequestContext) {
	var req getOrCreateLiveStreamRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误: " + err.Error()})
		return
	}
	if req.CreatorID <= 0 {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "creator_id 必填"})
		return
	}
	liveStream, err := h.productService.GetOrCreateLiveStream(ctx, req.CreatorID, req.CreatorName)
	if err != nil {
		c.JSON(500, map[string]interface{}{"code": 500, "message": "获取直播间失败: " + err.Error()})
		return
	}
	if !liveStream.IsActive() {
		c.JSON(409, map[string]interface{}{"code": 409, "message": "直播间已被禁用，无法创建竞拍"})
		return
	}
	c.JSON(200, map[string]interface{}{"code": 200, "message": "success", "data": liveStreamSummary{
		ID:         liveStream.ID,
		Name:       liveStream.Name,
		CoverImage: liveStream.CoverImage,
		Status:     int(liveStream.Status),
		CreatorID:  liveStream.CreatorID,
	}})
}
```

在 `ProductService` 增加代理方法：

```go
func (s *ProductService) GetOrCreateLiveStream(ctx context.Context, creatorID int64, creatorName string) (*model.LiveStream, error) {
	return s.liveStreamService.GetOrCreateLiveStream(ctx, creatorID, creatorName)
}
```

- [ ] **Step 5: 注册路由**

在 `backend/product/main.go` 的 internal routes 增加：

```go
internal.GET("/products/:id/auction-info", internalHandler.GetAuctionProductInfo)
internal.POST("/live-streams/get-or-create", internalHandler.GetOrCreateActiveLiveStream)
```

- [ ] **Step 6: 运行测试确认通过**

Run:

```bash
cd backend/product && go test ./handler -run 'TestInternalHandler_(GetAuctionProductInfo|GetOrCreateActiveLiveStream)' -count=1
```

Expected: PASS。

- [ ] **Step 7: 提交**

```bash
git add backend/product/service/product.go backend/product/handler/internal.go backend/product/main.go backend/product/handler/internal_test.go
git commit -m "feat(product): expose auction product facts internally"
```

---

### Task 3: Auction ProductClient 增加商品事实与直播间方法

**Files:**
- Modify: `backend/auction/client/product_client.go`
- Test: `backend/auction/client/product_client_test.go`

- [ ] **Step 1: 写失败测试**

在 `backend/auction/client/product_client_test.go` 增加：

```go
func TestHTTPProductClient_GetAuctionProductInfo(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		assert.Equal(t, "test-token", r.Header.Get("X-Internal-Token"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"message":"success","data":{"id":11,"owner_id":1001,"status":1,"rule_bound":true}}`))
	}))
	defer ts.Close()

	c := NewHTTPProductClient(ts.URL, time.Second)
	c.SetInternalToken("test-token")
	info, err := c.GetAuctionProductInfo(context.Background(), 11)

	require.NoError(t, err)
	assert.Equal(t, "/internal/products/11/auction-info", gotPath)
	assert.Equal(t, int64(11), info.ID)
	assert.Equal(t, int64(1001), info.OwnerID)
	assert.Equal(t, 1, info.Status)
	assert.True(t, info.RuleBound)
}

func TestHTTPProductClient_GetOrCreateActiveLiveStream(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/live-streams/get-or-create", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, float64(1001), req["creator_id"])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"message":"success","data":{"id":77,"creator_id":1001,"status":1,"name":"merchant_1001的直播间"}}`))
	}))
	defer ts.Close()

	c := NewHTTPProductClient(ts.URL, time.Second)
	live, err := c.GetOrCreateActiveLiveStream(context.Background(), 1001, "merchant_1001")

	require.NoError(t, err)
	assert.Equal(t, int64(77), live.ID)
	assert.Equal(t, int64(1001), live.CreatorID)
	assert.Equal(t, 1, live.Status)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run:

```bash
cd backend/auction && go test ./client -run 'TestHTTPProductClient_(GetAuctionProductInfo|GetOrCreateActiveLiveStream)' -count=1
```

Expected: FAIL，方法未定义。

- [ ] **Step 3: 实现 DTO、接口和 HTTP 方法**

在 `backend/auction/client/product_client.go` 增加类型：

```go
type AuctionProductInfo struct {
	ID        int64 `json:"id"`
	OwnerID   int64 `json:"owner_id"`
	Status    int   `json:"status"`
	RuleBound bool  `json:"rule_bound"`
}

type LiveStreamInfo struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatorID int64  `json:"creator_id"`
	Status    int    `json:"status"`
}
```

扩展 `ProductClient`：

```go
GetAuctionProductInfo(ctx context.Context, productID int64) (*AuctionProductInfo, error)
GetOrCreateActiveLiveStream(ctx context.Context, creatorID int64, creatorName string) (*LiveStreamInfo, error)
```

新增响应结构：

```go
type internalAuctionProductInfoResponse struct {
	Code    int                `json:"code"`
	Data    AuctionProductInfo `json:"data"`
	Message string             `json:"message"`
}

type internalLiveStreamResponse struct {
	Code    int            `json:"code"`
	Data    LiveStreamInfo `json:"data"`
	Message string         `json:"message"`
}
```

新增方法：

```go
func (c *HTTPProductClient) GetAuctionProductInfo(ctx context.Context, productID int64) (*AuctionProductInfo, error) {
	endpoint := fmt.Sprintf("%s/internal/products/%d/auction-info", c.baseURL, productID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if c.internalToken != "" {
		req.Header.Set("X-Internal-Token", c.internalToken)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call product-service: %w", err)
	}
	defer resp.Body.Close()
	defer io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("product-service returned status %d", resp.StatusCode)
	}
	var body internalAuctionProductInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if body.Code != 0 && body.Code != http.StatusOK {
		return nil, fmt.Errorf("product-service business code %d: %s", body.Code, body.Message)
	}
	return &body.Data, nil
}

func (c *HTTPProductClient) GetOrCreateActiveLiveStream(ctx context.Context, creatorID int64, creatorName string) (*LiveStreamInfo, error) {
	payload, err := json.Marshal(map[string]interface{}{"creator_id": creatorID, "creator_name": creatorName})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	endpoint := c.baseURL + "/internal/live-streams/get-or-create"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.internalToken != "" {
		req.Header.Set("X-Internal-Token", c.internalToken)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call product-service: %w", err)
	}
	defer resp.Body.Close()
	defer io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("product-service returned status %d", resp.StatusCode)
	}
	var body internalLiveStreamResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if body.Code != 0 && body.Code != http.StatusOK {
		return nil, fmt.Errorf("product-service business code %d: %s", body.Code, body.Message)
	}
	return &body.Data, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run:

```bash
cd backend/auction && go test ./client -run 'TestHTTPProductClient_(GetAuctionProductInfo|GetOrCreateActiveLiveStream)' -count=1
```

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add backend/auction/client/product_client.go backend/auction/client/product_client_test.go
git commit -m "feat(auction): extend product client for auction creation"
```

---

### Task 4: Auction DAO 支持活跃唯一查询与 MySQL 兜底索引

**Files:**
- Modify: `backend/auction/dao/auction.go`
- Create: `backend/auction/dao/auction_schema.go`
- Modify: `backend/auction/main.go`
- Test: `backend/auction/dao/auction_test.go`
- Test: `backend/auction/dao/auction_schema_test.go`

- [ ] **Step 1: 写失败测试：active/latest 查询**

在 `backend/auction/dao/auction_test.go` 增加：

```go
func TestAuctionDAO_GetActiveAndLatestTerminalByProductID(t *testing.T) {
	db := newAuctionDAOTestDB(t)
	d := NewAuctionDAO(db)
	now := time.Now()
	winnerID := int64(2001)
	rows := []model.Auction{
		{ID: 1, ProductID: 11, Status: model.AuctionStatusEnded, WinnerID: nil, StartTime: now.Add(-4 * time.Hour), EndTime: now.Add(-3 * time.Hour)},
		{ID: 2, ProductID: 11, Status: model.AuctionStatusEnded, WinnerID: &winnerID, StartTime: now.Add(-2 * time.Hour), EndTime: now.Add(-time.Hour)},
		{ID: 3, ProductID: 11, Status: model.AuctionStatusPending, StartTime: now, EndTime: now.Add(time.Hour)},
	}
	require.NoError(t, db.Create(&rows).Error)

	active, err := d.GetActiveByProductID(context.Background(), 11)
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, int64(3), active.ID)

	terminal, err := d.GetLatestTerminalByProductID(context.Background(), 11)
	require.NoError(t, err)
	require.NotNil(t, terminal)
	assert.Equal(t, int64(2), terminal.ID)
	assert.NotNil(t, terminal.WinnerID)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run:

```bash
cd backend/auction && go test ./dao -run TestAuctionDAO_GetActiveAndLatestTerminalByProductID -count=1
```

Expected: FAIL，DAO 方法未定义。

- [ ] **Step 3: 实现 DAO 查询**

在 `backend/auction/dao/auction.go` 增加：

```go
func (d *AuctionDAO) GetActiveByProductID(ctx context.Context, productID int64) (*model.Auction, error) {
	var auction model.Auction
	err := d.db.WithContext(ctx).
		Where("product_id = ? AND status IN ?", productID, []model.AuctionStatus{
			model.AuctionStatusPending,
			model.AuctionStatusOngoing,
			model.AuctionStatusDelayed,
		}).
		Order("id DESC").
		First(&auction).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &auction, nil
}

func (d *AuctionDAO) GetLatestTerminalByProductID(ctx context.Context, productID int64) (*model.Auction, error) {
	var auction model.Auction
	err := d.db.WithContext(ctx).
		Where("product_id = ? AND status IN ?", productID, []model.AuctionStatus{
			model.AuctionStatusEnded,
			model.AuctionStatusCancelled,
		}).
		Order("end_time DESC, id DESC").
		First(&auction).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &auction, nil
}
```

- [ ] **Step 4: 写 MySQL DDL helper**

创建 `backend/auction/dao/auction_schema.go`：

```go
package dao

import (
	"strings"

	"gorm.io/gorm"
)

func EnsureAuctionActiveProductUniqueIndex(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	dialect := db.Dialector.Name()
	if dialect != "mysql" {
		return nil
	}

	var columnCount int64
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'auctions'
		  AND COLUMN_NAME = 'active_product_key'
	`).Scan(&columnCount).Error; err != nil {
		return err
	}
	if columnCount == 0 {
		if err := db.Exec(`
			ALTER TABLE auctions
			  ADD COLUMN active_product_key BIGINT AS
			    (CASE WHEN status IN (0,1,2) THEN product_id ELSE NULL END) STORED
		`).Error; err != nil && !strings.Contains(err.Error(), "Duplicate column") {
			return err
		}
	}

	var indexCount int64
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'auctions'
		  AND INDEX_NAME = 'uk_active_product'
	`).Scan(&indexCount).Error; err != nil {
		return err
	}
	if indexCount == 0 {
		if err := db.Exec(`ALTER TABLE auctions ADD UNIQUE KEY uk_active_product (active_product_key)`).Error; err != nil && !strings.Contains(err.Error(), "Duplicate key name") {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 5: main.go 接入 DDL helper**

在 `backend/auction/main.go` 的 `AutoMigrate` 后增加：

```go
if err := dao.EnsureAuctionActiveProductUniqueIndex(db); err != nil {
	log.Printf("Warning: ensure active product unique index failed: %v", err)
}
```

- [ ] **Step 6: 运行 DAO 测试**

Run:

```bash
cd backend/auction && go test ./dao -run 'TestAuctionDAO_GetActiveAndLatestTerminalByProductID|TestEnsureAuctionActiveProductUniqueIndex' -count=1
```

Expected: PASS。sqlite 环境下 DDL helper 返回 nil，不执行 MySQL DDL。

- [ ] **Step 7: 提交**

```bash
git add backend/auction/dao/auction.go backend/auction/dao/auction_schema.go backend/auction/main.go backend/auction/dao/auction_test.go backend/auction/dao/auction_schema_test.go
git commit -m "feat(auction): enforce active auction uniqueness at storage boundary"
```

---

### Task 5: AuctionService 创建竞拍 Fail-closed 校验

**Files:**
- Modify: `backend/auction/service/auction.go`
- Test: `backend/auction/service/auction_create_test.go`

- [ ] **Step 1: 写失败测试：创建校验矩阵**

创建 `backend/auction/service/auction_create_test.go`：

```go
package service

import (
	"context"
	"testing"
	"time"

	"auction-service/dao"
	"auction-service/model"

	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAuctionService_CreateAuctionValidatesProductLifecycle(t *testing.T) {
	tests := []struct {
		name        string
		req         CreateAuctionRequest
		seed        func(*gorm.DB)
		wantErr     string
		wantCreated bool
	}{
		{
			name: "merchant can create auction for own published product with bound rule",
			req: CreateAuctionRequest{ProductID: 11, CreatorID: ptrInt64(1001), Duration: 3600, ProductOwnerID: 1001, ProductStatus: 1, RuleBound: true, LiveStreamID: 77},
			wantCreated: true,
		},
		{
			name: "draft product rejected",
			req: CreateAuctionRequest{ProductID: 11, CreatorID: ptrInt64(1001), Duration: 3600, ProductOwnerID: 1001, ProductStatus: 0, RuleBound: true, LiveStreamID: 77},
			wantErr: "商品未进入竞拍池",
		},
		{
			name: "other merchant product rejected",
			req: CreateAuctionRequest{ProductID: 11, CreatorID: ptrInt64(1001), Duration: 3600, ProductOwnerID: 2002, ProductStatus: 1, RuleBound: true, LiveStreamID: 77},
			wantErr: "商品不存在或不属于当前商家",
		},
		{
			name: "missing rule rejected",
			req: CreateAuctionRequest{ProductID: 11, CreatorID: ptrInt64(1001), Duration: 3600, ProductOwnerID: 1001, ProductStatus: 1, RuleBound: false, LiveStreamID: 77},
			wantErr: "规则模板不存在或不属于当前商家",
		},
		{
			name: "active auction rejected",
			req: CreateAuctionRequest{ProductID: 11, CreatorID: ptrInt64(1001), Duration: 3600, ProductOwnerID: 1001, ProductStatus: 1, RuleBound: true, LiveStreamID: 77},
			seed: func(db *gorm.DB) {
				require.NoError(t, db.Create(&model.Auction{ProductID: 11, Status: model.AuctionStatusOngoing, StartTime: time.Now().Add(-time.Minute), EndTime: time.Now().Add(time.Hour)}).Error)
			},
			wantErr: "该商品已有待开始或进行中的竞拍场次",
		},
		{
			name: "latest sold auction rejected",
			req: CreateAuctionRequest{ProductID: 11, CreatorID: ptrInt64(1001), Duration: 3600, ProductOwnerID: 1001, ProductStatus: 1, RuleBound: true, LiveStreamID: 77},
			seed: func(db *gorm.DB) {
				winnerID := int64(3003)
				require.NoError(t, db.Create(&model.Auction{ProductID: 11, Status: model.AuctionStatusEnded, WinnerID: &winnerID, StartTime: time.Now().Add(-2 * time.Hour), EndTime: time.Now().Add(-time.Hour)}).Error)
			},
			wantErr: "已成交商品不能再次创建竞拍",
		},
		{
			name: "latest unsold auction allows retry",
			req: CreateAuctionRequest{ProductID: 11, CreatorID: ptrInt64(1001), Duration: 3600, ProductOwnerID: 1001, ProductStatus: 1, RuleBound: true, LiveStreamID: 77},
			seed: func(db *gorm.DB) {
				require.NoError(t, db.Create(&model.Auction{ProductID: 11, Status: model.AuctionStatusEnded, WinnerID: nil, StartTime: time.Now().Add(-2 * time.Hour), EndTime: time.Now().Add(-time.Hour)}).Error)
			},
			wantCreated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newAuctionCreateTestDB(t)
			if tt.seed != nil {
				tt.seed(db)
			}
			svc := NewAuctionService(dao.NewAuctionDAO(db))
			got, err := svc.CreateAuction(context.Background(), &tt.req)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, int64(11), got.ProductID)
			assert.Equal(t, model.AuctionStatusPending, got.Status)
			assert.Equal(t, int64(77), *got.LiveStreamID)
			assert.True(t, got.CurrentPrice.Equal(decimal.Zero))
		})
	}
}

func newAuctionCreateTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.Auction{}))
	return db
}
```

- [ ] **Step 2: 运行测试确认失败**

Run:

```bash
cd backend/auction && go test ./service -run TestAuctionService_CreateAuctionValidatesProductLifecycle -count=1
```

Expected: FAIL，`CreateAuctionRequest` 字段不存在，校验未实现。

- [ ] **Step 3: 实现请求结构和校验**

将 `backend/auction/service/auction.go` 的 `CreateAuctionRequest` 替换为：

```go
type CreateAuctionRequest struct {
	ProductID      int64
	CreatorID      *int64
	Duration       int
	ProductOwnerID int64
	ProductStatus  int
	RuleBound      bool
	LiveStreamID   int64
}
```

在 `service/auction.go` 增加错误变量：

```go
var (
	ErrProductOwnershipMismatch    = errors.New("商品不存在或不属于当前商家")
	ErrProductNotSchedulable       = errors.New("商品未进入竞拍池")
	ErrAuctionRuleNotBound         = errors.New("规则模板不存在或不属于当前商家")
	ErrActiveAuctionExists         = errors.New("该商品已有待开始或进行中的竞拍场次")
	ErrSoldProductCannotBeReauctioned = errors.New("已成交商品不能再次创建竞拍")
)
```

> 身份/角色校验（未登录、非商家）只在 handler 层做（见 Task 6），service 不重复拦截，避免不可达的死分支。service 仅负责业务不变量校验。

将 `CreateAuction` 替换为：

```go
func (s *AuctionService) CreateAuction(ctx context.Context, req *CreateAuctionRequest) (*model.Auction, error) {
	if req == nil {
		return nil, errors.New("创建竞拍请求不能为空")
	}
	if req.CreatorID == nil || *req.CreatorID <= 0 {
		return nil, errors.New("创建者ID非法")
	}
	if req.ProductID <= 0 {
		return nil, errors.New("商品ID非法")
	}
	if req.Duration <= 0 {
		return nil, errors.New("竞拍时长必须大于0")
	}
	if req.ProductOwnerID != *req.CreatorID {
		return nil, ErrProductOwnershipMismatch
	}
	if req.ProductStatus != 1 {
		return nil, ErrProductNotSchedulable
	}
	if !req.RuleBound {
		return nil, ErrAuctionRuleNotBound
	}
	if req.LiveStreamID <= 0 {
		return nil, errors.New("直播间不可用")
	}

	active, err := s.auctionDAO.GetActiveByProductID(ctx, req.ProductID)
	if err != nil {
		return nil, err
	}
	if active != nil {
		return nil, ErrActiveAuctionExists
	}

	latest, err := s.auctionDAO.GetLatestTerminalByProductID(ctx, req.ProductID)
	if err != nil {
		return nil, err
	}
	if latest != nil && latest.Status == model.AuctionStatusEnded && latest.WinnerID != nil {
		return nil, ErrSoldProductCannotBeReauctioned
	}

	now := time.Now()
	liveStreamID := req.LiveStreamID
	auction := &model.Auction{
		ProductID:    req.ProductID,
		LiveStreamID: &liveStreamID,
		CreatorID:    req.CreatorID,
		Status:       model.AuctionStatusPending,
		CurrentPrice: decimal.Zero,
		StartTime:    now,
		EndTime:      now.Add(time.Duration(req.Duration) * time.Second),
		DelayUsed:    0,
	}
	if err := s.auctionDAO.Create(ctx, auction); err != nil {
		if strings.Contains(err.Error(), "uk_active_product") || strings.Contains(err.Error(), "Duplicate entry") {
			return nil, ErrActiveAuctionExists
		}
		return nil, err
	}
	return auction, nil
}
```

`service/auction.go` import 增加 `strings`。

- [ ] **Step 4: 运行测试确认通过**

Run:

```bash
cd backend/auction && go test ./service -run TestAuctionService_CreateAuctionValidatesProductLifecycle -count=1
```

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add backend/auction/service/auction.go backend/auction/service/auction_create_test.go
git commit -m "feat(auction): validate product lifecycle before creating auctions"
```

---

### Task 6: Auction Create Handler 编排 product-service 与业务错误码

**Files:**
- Modify: `backend/auction/handler/auction.go`
- Test: `backend/auction/handler/auction_create_test.go`

- [ ] **Step 1: 写失败测试**

在 `backend/auction/handler/auction_create_test.go` 增加：

```go
func TestAuctionHandler_CreateRequiresMerchantAndWritesLiveStream(t *testing.T) {
	db := newAuctionHandlerCreateTestDB(t)
	auctionDAO := dao.NewAuctionDAO(db)
	svc := service.NewAuctionService(auctionDAO)
	h := NewAuctionHandler(svc)
	h.SetProductClient(&fakeCreateProductClient{
		info: &client.AuctionProductInfo{ID: 11, OwnerID: 1001, Status: 1, RuleBound: true},
		live: &client.LiveStreamInfo{ID: 77, CreatorID: 1001, Status: 1},
	})

	app := server.Default()
	app.POST("/api/v1/auctions", h.Create)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auctions", strings.NewReader(`{"product_id":11,"duration":3600}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "1001")
	req.Header.Set("X-User-Role", "merchant")
	resp := performRequest(app, req)

	require.Equal(t, http.StatusCreated, resp.Code)
	var auction model.Auction
	require.NoError(t, db.First(&auction, "product_id = ?", 11).Error)
	require.NotNil(t, auction.LiveStreamID)
	assert.Equal(t, int64(77), *auction.LiveStreamID)
}

func TestAuctionHandler_CreateRejectsUserRole(t *testing.T) {
	db := newAuctionHandlerCreateTestDB(t)
	h := NewAuctionHandler(service.NewAuctionService(dao.NewAuctionDAO(db)))
	app := server.Default()
	app.POST("/api/v1/auctions", h.Create)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auctions", strings.NewReader(`{"product_id":11,"duration":3600}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "1001")
	req.Header.Set("X-User-Role", "user")
	resp := performRequest(app, req)

	assert.Equal(t, http.StatusForbidden, resp.Code)
}
```

补充 fake：

```go
type fakeCreateProductClient struct {
	info *client.AuctionProductInfo
	live *client.LiveStreamInfo
	err  error
}

func (f *fakeCreateProductClient) ListProductIDsByCategory(ctx context.Context, categoryID int64) ([]int64, error) {
	return nil, nil
}

func (f *fakeCreateProductClient) BatchGetSummaries(ctx context.Context, ids []int64) (map[int64]client.ProductSummary, error) {
	return map[int64]client.ProductSummary{}, nil
}

func (f *fakeCreateProductClient) GetAuctionProductInfo(ctx context.Context, productID int64) (*client.AuctionProductInfo, error) {
	return f.info, f.err
}

func (f *fakeCreateProductClient) GetOrCreateActiveLiveStream(ctx context.Context, creatorID int64, creatorName string) (*client.LiveStreamInfo, error) {
	return f.live, f.err
}
```

- [ ] **Step 2: 运行测试确认失败**

Run:

```bash
cd backend/auction && go test ./handler -run 'TestAuctionHandler_Create' -count=1
```

Expected: FAIL，handler 仍直接构造旧 `CreateAuctionRequest`。

- [ ] **Step 3: 修改 handler 请求体与编排**

将 `CreateAuctionRequest` 替换为：

```go
type CreateAuctionRequest struct {
	ProductID int64 `json:"product_id" binding:"required"`
	Duration  int   `json:"duration" binding:"required"`
}
```

将 `Create` 中构造 service 请求的逻辑替换为：

```go
creatorID, ok := userIDFromHeader(c)
if !ok {
	c.JSON(401, map[string]interface{}{"code": 401, "message": "未认证，请先登录"})
	return
}
role := string(c.GetHeader("X-User-Role"))
if role != merchantRole {
	c.JSON(403, map[string]interface{}{"code": 403, "message": "权限不足"})
	return
}
if h.productClient == nil {
	c.JSON(500, map[string]interface{}{"code": 500, "message": "商品服务不可用"})
	return
}

var req CreateAuctionRequest
if err := c.BindJSON(&req); err != nil {
	c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误: " + err.Error()})
	return
}

info, err := h.productClient.GetAuctionProductInfo(ctx, req.ProductID)
if err != nil {
	c.JSON(400, map[string]interface{}{"code": 400, "message": "商品不存在或不属于当前商家"})
	return
}
live, err := h.productClient.GetOrCreateActiveLiveStream(ctx, creatorID, "merchant_"+strconv.FormatInt(creatorID, 10))
if err != nil {
	c.JSON(409, map[string]interface{}{"code": 409, "message": "直播间已被禁用，无法创建竞拍"})
	return
}

auction, err := h.auctionService.CreateAuction(ctx, &service.CreateAuctionRequest{
	ProductID:      req.ProductID,
	CreatorID:      &creatorID,
	Duration:       req.Duration,
	ProductOwnerID: info.OwnerID,
	ProductStatus:  info.Status,
	RuleBound:      info.RuleBound,
	LiveStreamID:   live.ID,
})
if err != nil {
	writeCreateAuctionError(c, err)
	return
}
c.JSON(201, auction)
```

新增 helper：

```go
func writeCreateAuctionError(c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, service.ErrActiveAuctionExists):
		c.JSON(409, map[string]interface{}{"code": 409, "message": err.Error()})
	case errors.Is(err, service.ErrProductOwnershipMismatch),
		errors.Is(err, service.ErrProductNotSchedulable),
		errors.Is(err, service.ErrAuctionRuleNotBound),
		errors.Is(err, service.ErrSoldProductCannotBeReauctioned):
		c.JSON(400, map[string]interface{}{"code": 400, "message": err.Error()})
	default:
		c.JSON(500, map[string]interface{}{"code": 500, "message": "创建竞拍失败: " + err.Error()})
	}
}
```

> 身份/角色校验已在 `Create` 入口完成（401/403），service 不再返回身份类错误，故此处不映射 401/403。

- [ ] **Step 3.1: 清理 handler 未使用的 import**

旧 `Create` 用了 `time.Now()`，替换后该用法被删除。检查 `backend/auction/handler/auction.go` 是否还有其它 `time.` 引用；若没有，从 import 块删除 `"time"`，否则 `go build` 会报 `imported and not used`。同理确认 `strconv`（新逻辑 `strconv.FormatInt` 仍在用）保留。

Run 验证：

```bash
cd backend/auction && go build ./handler
```

Expected: 编译通过，无 unused import 报错。

- [ ] **Step 4: 运行测试确认通过**

Run:

```bash
cd backend/auction && go test ./handler -run 'TestAuctionHandler_Create' -count=1
```

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add backend/auction/handler/auction.go backend/auction/handler/auction_create_test.go
git commit -m "feat(auction): orchestrate product checks when creating auctions"
```

---

### Task 7: Product AdminList 返回派生展示状态

**Files:**
- Create: `backend/auction/handler/internal_product_auctions.go`
- Modify: `backend/auction/main.go`
- Modify: `backend/product/client/auction_client.go`
- Modify: `backend/product/handler/product.go`
- Modify: `backend/product/main.go`
- Test: `backend/auction/handler/internal_product_auctions_test.go`
- Test: `backend/product/handler/product_test.go`

- [ ] **Step 1: 写失败测试：auction-service by-products 内部接口**

创建 `backend/auction/handler/internal_product_auctions_test.go`：

```go
func TestInternalProductAuctionsHandler_ByProducts(t *testing.T) {
	db := newAuctionHandlerCreateTestDB(t)
	now := time.Now()
	winnerID := int64(2001)
	require.NoError(t, db.Create(&[]model.Auction{
		{ID: 1, ProductID: 11, Status: model.AuctionStatusOngoing, StartTime: now.Add(-time.Minute), EndTime: now.Add(time.Hour)},
		{ID: 2, ProductID: 12, Status: model.AuctionStatusEnded, WinnerID: &winnerID, StartTime: now.Add(-2 * time.Hour), EndTime: now.Add(-time.Hour)},
		{ID: 3, ProductID: 13, Status: model.AuctionStatusEnded, WinnerID: nil, StartTime: now.Add(-2 * time.Hour), EndTime: now.Add(-time.Hour)},
	}).Error)

	h := NewInternalProductAuctionsHandler(dao.NewAuctionDAO(db))
	app := server.Default()
	app.POST("/internal/auctions/by-products", h.Handle)
	req := httptest.NewRequest(http.MethodPost, "/internal/auctions/by-products", strings.NewReader(`{"product_ids":[11,12,13,14]}`))
	req.Header.Set("Content-Type", "application/json")
	resp := performRequest(app, req)

	require.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), `"product_id":11`)
	assert.Contains(t, resp.Body.String(), `"active_auction_id":1`)
	assert.Contains(t, resp.Body.String(), `"latest_auction_result":"sold"`)
	assert.Contains(t, resp.Body.String(), `"latest_auction_result":"unsold"`)
}
```

- [ ] **Step 2: 实现 auction internal handler**

创建 `backend/auction/handler/internal_product_auctions.go`：

```go
package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"auction-service/dao"
	"auction-service/model"
)

type productAuctionsDAO interface {
	GetActiveByProductID(ctx context.Context, productID int64) (*model.Auction, error)
	GetLatestTerminalByProductID(ctx context.Context, productID int64) (*model.Auction, error)
}

type InternalProductAuctionsHandler struct {
	dao productAuctionsDAO
}

type productAuctionsRequest struct {
	ProductIDs []int64 `json:"product_ids"`
}

type productAuctionState struct {
	ProductID           int64  `json:"product_id"`
	ActiveAuctionID     *int64 `json:"active_auction_id,omitempty"`
	ActiveStatus        *int   `json:"active_status,omitempty"`
	LatestAuctionID     *int64 `json:"latest_auction_id,omitempty"`
	LatestAuctionStatus *int   `json:"latest_auction_status,omitempty"`
	LatestAuctionResult string `json:"latest_auction_result,omitempty"`
}

func NewInternalProductAuctionsHandler(auctionDAO *dao.AuctionDAO) *InternalProductAuctionsHandler {
	return &InternalProductAuctionsHandler{dao: auctionDAO}
}

func (h *InternalProductAuctionsHandler) Handle(ctx context.Context, c *app.RequestContext) {
	var req productAuctionsRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误: " + err.Error()})
		return
	}
	if len(req.ProductIDs) == 0 {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "product_ids 不能为空"})
		return
	}
	items := make([]productAuctionState, 0, len(req.ProductIDs))
	seen := make(map[int64]struct{}, len(req.ProductIDs))
	for _, productID := range req.ProductIDs {
		if productID <= 0 {
			continue
		}
		if _, ok := seen[productID]; ok {
			continue
		}
		seen[productID] = struct{}{}
		item := productAuctionState{ProductID: productID}
		active, err := h.dao.GetActiveByProductID(ctx, productID)
		if err != nil {
			c.JSON(500, map[string]interface{}{"code": 500, "message": "查询活跃竞拍失败: " + err.Error()})
			return
		}
		if active != nil {
			status := int(active.Status)
			item.ActiveAuctionID = &active.ID
			item.ActiveStatus = &status
		}
		latest, err := h.dao.GetLatestTerminalByProductID(ctx, productID)
		if err != nil {
			c.JSON(500, map[string]interface{}{"code": 500, "message": "查询最近竞拍失败: " + err.Error()})
			return
		}
		if latest != nil {
			status := int(latest.Status)
			item.LatestAuctionID = &latest.ID
			item.LatestAuctionStatus = &status
			if latest.Status == model.AuctionStatusEnded && latest.WinnerID != nil {
				item.LatestAuctionResult = "sold"
			}
			if latest.Status == model.AuctionStatusEnded && latest.WinnerID == nil {
				item.LatestAuctionResult = "unsold"
			}
		}
		items = append(items, item)
	}
	c.JSON(200, map[string]interface{}{"code": 200, "message": "success", "data": map[string]interface{}{"items": items}})
}
```

注册路由：

```go
productAuctionsHandler := handler.NewInternalProductAuctionsHandler(auctionDAO)
```

并将其传入 `registerInternalRoutes`，在内部路由增加：

```go
internal.POST("/auctions/by-products", productAuctionsHandler.Handle)
```

- [ ] **Step 3: product-service AuctionClient 解码 by-products**

在 `backend/product/client/auction_client.go` 增加：

```go
type ProductAuctionState struct {
	ProductID           int64  `json:"product_id"`
	ActiveAuctionID     *int64 `json:"active_auction_id"`
	ActiveStatus        *int   `json:"active_status"`
	LatestAuctionID     *int64 `json:"latest_auction_id"`
	LatestAuctionStatus *int   `json:"latest_auction_status"`
	LatestAuctionResult string `json:"latest_auction_result"`
}

func (c *AuctionClient) BatchProductAuctionStates(ctx context.Context, productIDs []int64) (map[int64]ProductAuctionState, error) {
	if len(productIDs) == 0 {
		return map[int64]ProductAuctionState{}, nil
	}
	payload, err := json.Marshal(map[string]interface{}{"product_ids": productIDs})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	reqURL := fmt.Sprintf("%s/internal/auctions/by-products", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.internalToken != "" {
		req.Header.Set("X-Internal-Token", c.internalToken)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call auction-service: %w", err)
	}
	defer resp.Body.Close()
	defer io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auction-service returned status %d", resp.StatusCode)
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Items []ProductAuctionState `json:"items"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if body.Code != 0 && body.Code != 200 {
		return nil, fmt.Errorf("auction-service business code %d: %s", body.Code, body.Message)
	}
	result := make(map[int64]ProductAuctionState, len(body.Data.Items))
	for _, item := range body.Data.Items {
		result[item.ProductID] = item
	}
	return result, nil
}
```

- [ ] **Step 4: ProductHandler AdminList 包装派生状态**

> 先读现有 `backend/product/handler/product.go` 的 `AdminList`（约 L161 起），确认列表变量的实际名称与类型（是否为 `[]model.Product`、是否已直接 `c.JSON` 返回 `list`）。下方 `buildAdminProductItems(products, states)` 假设列表变量名为 `products`、类型 `[]model.Product`；若现状不同，按现状调整入参，不要照抄变量名。还需在 `AdminList` 中用注入的 provider 拉取 states：若 `h.auctionStateProvider == nil`（单测或未接线）则 states 用空 map 兜底（Fail-open 仅限展示派生，不影响核心校验）。

在 `backend/product/handler/product.go` 给 `ProductHandler` 增加字段和 setter：

```go
type productAuctionStateProvider interface {
	BatchProductAuctionStates(ctx context.Context, productIDs []int64) (map[int64]client.ProductAuctionState, error)
}

func (h *ProductHandler) SetAuctionStateProvider(provider productAuctionStateProvider) {
	h.auctionStateProvider = provider
}
```

新增 DTO：

```go
type adminProductItem struct {
	model.Product
	DisplayStatus      string `json:"display_status"`
	DisplayStatusLabel string `json:"display_status_label"`
	ActiveAuctionID    *int64 `json:"active_auction_id,omitempty"`
	LatestAuctionID    *int64 `json:"latest_auction_id,omitempty"`
	LatestAuctionResult string `json:"latest_auction_result,omitempty"`
}
```

将 `AdminList` 返回 list 改为：

```go
items := buildAdminProductItems(products, states)
c.JSON(200, map[string]interface{}{
	"code":    200,
	"message": "success",
	"data": map[string]interface{}{
		"list":      items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	},
})
```

新增 builder：

```go
func buildAdminProductItems(products []model.Product, states map[int64]client.ProductAuctionState) []adminProductItem {
	items := make([]adminProductItem, 0, len(products))
	for _, p := range products {
		item := adminProductItem{Product: p}
		state, ok := states[p.ID]
		if ok {
			item.ActiveAuctionID = state.ActiveAuctionID
			item.LatestAuctionID = state.LatestAuctionID
			item.LatestAuctionResult = state.LatestAuctionResult
		}
		switch {
		case ok && state.ActiveAuctionID != nil:
			item.DisplayStatus = "auctioning"
			item.DisplayStatusLabel = "竞拍中"
		case ok && state.LatestAuctionResult == "sold":
			item.DisplayStatus = "sold"
			item.DisplayStatusLabel = "已拍卖"
		case ok && state.LatestAuctionResult == "unsold":
			item.DisplayStatus = "unsold"
			item.DisplayStatusLabel = "流拍"
		case p.Status == model.ProductStatusPublished:
			item.DisplayStatus = "schedulable"
			item.DisplayStatusLabel = "可排期"
		case p.Status == model.ProductStatusDraft:
			item.DisplayStatus = "draft"
			item.DisplayStatusLabel = "草稿"
		case p.Status == model.ProductStatusUnpublished:
			item.DisplayStatus = "unpublished"
			item.DisplayStatusLabel = "已下架"
		default:
			item.DisplayStatus = "unknown"
			item.DisplayStatusLabel = "未知"
		}
		items = append(items, item)
	}
	return items
}
```

- [ ] **Step 5: main.go 注入 ProductHandler**

在 `backend/product/main.go` 现有 `auctionClient := client.NewAuctionClient(auctionSvcURL, 2*time.Second)` 后增加：

```go
productHandler.SetAuctionStateProvider(auctionClient)
```

- [ ] **Step 6: 运行测试**

Run:

```bash
cd backend/auction && go test ./handler -run TestInternalProductAuctionsHandler_ByProducts -count=1
cd backend/product && go test ./handler -run TestProductHandler_AdminListDerivedStatus -count=1
```

Expected: PASS。

- [ ] **Step 7: 提交**

```bash
git add backend/auction/handler/internal_product_auctions.go backend/auction/main.go backend/auction/handler/internal_product_auctions_test.go backend/product/client/auction_client.go backend/product/handler/product.go backend/product/main.go backend/product/handler/product_test.go
git commit -m "feat(product): derive admin product auction status"
```

---

### Task 8: Admin 前端文案、筛选与错误展示

**Files:**
- Modify: `frontend/admin/src/shared/api/types.ts`
- Modify: `frontend/admin/src/pages-new/GoodsList.tsx`
- Modify: `frontend/admin/src/pages-new/AuctionList.tsx`
- Test: `frontend/admin/src/pages-new/__tests__/GoodsList.test.tsx`
- Test: `frontend/admin/src/pages-new/__tests__/AuctionList.createAuction.test.tsx`

- [ ] **Step 1: 写失败测试：商品列表文案**

在 `GoodsList.test.tsx` 增加断言：

```tsx
it("shows schedulable wording instead of publish wording", async () => {
  mockProductList([{ id: 11, name: "青花瓷", status: 0, display_status_label: "草稿", created_at: new Date().toISOString(), images: [] }])
  render(<GoodsList />)

  expect(await screen.findByText("草稿")).toBeInTheDocument()
  expect(screen.getByTitle("设为可排期")).toBeInTheDocument()
  expect(screen.queryByTitle("发布")).not.toBeInTheDocument()
})
```

- [ ] **Step 2: 写失败测试：竞拍创建空态与请求体**

在 `AuctionList.createAuction.test.tsx` 增加：

```tsx
it("shows empty schedulable products state", async () => {
  mockProductList([])
  mockTemplateList([{ id: 1, name: "默认模板", duration: 3600, is_default: true }])
  render(<AuctionList />)

  await userEvent.click(screen.getByRole("button", { name: "创建竞拍场次" }))

  expect(await screen.findByText("暂无可排期商品")).toBeInTheDocument()
})

it("creates auction without template_id live_stream_id or start_time", async () => {
  mockProductList([{ id: 11, name: "青花瓷", status: 1, display_status_label: "可排期", created_at: new Date().toISOString(), images: [] }])
  mockTemplateList([{ id: 1, name: "默认模板", duration: 3600, is_default: true }])
  const createSpy = mockAuctionCreate()
  render(<AuctionList />)

  await userEvent.click(screen.getByRole("button", { name: "创建竞拍场次" }))
  await userEvent.click(await screen.findByRole("button", { name: "确认创建竞拍" }))

  expect(createSpy).toHaveBeenCalledWith({ product_id: 11, duration: 3600 })
})
```

- [ ] **Step 3: 扩展 Product 类型**

在 `frontend/admin/src/shared/api/types.ts` 的 `Product` 增加：

```ts
display_status?: "auctioning" | "sold" | "unsold" | "schedulable" | "draft" | "unpublished" | "unknown";
display_status_label?: string;
active_auction_id?: number;
latest_auction_id?: number;
latest_auction_result?: "sold" | "unsold";
```

- [ ] **Step 4: GoodsList 文案与状态**

在 `GoodsList.tsx` 中：

```tsx
const statusMap: Record<number, { label: string; variant: BadgeProps["variant"] }> = {
  0: { label: "草稿", variant: "secondary" },
  1: { label: "可排期", variant: "success" },
  2: { label: "已下架", variant: "outline" },
}
```

Badge 文案替换为：

```tsx
{item.display_status_label || statusMap[item.status]?.label || "未知"}
```

发布按钮 title 替换为：

```tsx
title="设为可排期"
```

`handlePublish` 错误日志文案替换为：

```tsx
console.error("设为可排期失败:", e)
```

- [ ] **Step 5: AuctionList 筛选与 Badge**

新增 Badge 函数：

```tsx
function getAuctionStatus(auction: any): { label: string; variant: BadgeProps["variant"] } {
  if (auction.status === 0) return { label: "待开始", variant: "blue" }
  if (auction.status === 1) return { label: "竞拍中", variant: "success" }
  if (auction.status === 2) return { label: "竞拍中（延时）", variant: "warning" }
  if (auction.status === 3 && auction.winner_id) return { label: "已拍卖", variant: "outline" }
  if (auction.status === 3 && !auction.winner_id) return { label: "流拍", variant: "secondary" }
  if (auction.status === 4) return { label: "已取消", variant: "secondary" }
  return { label: "未知", variant: "secondary" }
}
```

Tabs 替换为：

```tsx
<TabsTrigger value="all">全部场次</TabsTrigger>
<TabsTrigger value="active">竞拍中</TabsTrigger>
<TabsTrigger value="sold">已拍卖</TabsTrigger>
<TabsTrigger value="unsold">流拍</TabsTrigger>
<TabsTrigger value="cancelled">已取消</TabsTrigger>
```

`handleStatusChange` 改为先用现有 `status` 参数表达能表达的筛选：

```tsx
const statusValue = { active: 1, sold: 3, unsold: 3, cancelled: 4 }[value]
setStatusFilter(statusValue)
```

> **已知限制（本次范围外）：** spec §3.2 要求「已拍卖/流拍」区分展示，但后端 `AdminList` 当前只支持按 `status` 过滤，无法区分 `status=3 AND winner_id IS [NOT] NULL`。本次仅做前端 tab 文案对齐，`sold`/`unsold` 两个 tab 暂都映射 `status=3`，前端拉到列表后再按 `winner_id` 做客户端二次过滤展示 Badge。后端聚合筛选（`result=sold/unsold` query 参数）留待后续迭代，不在本计划范围。执行者需在 `AuctionList.tsx` 增加客户端过滤：

```tsx
const visibleAuctions = auctions.filter((a) => {
  if (activeTab === "sold") return a.status === 3 && !!a.winner_id
  if (activeTab === "unsold") return a.status === 3 && !a.winner_id
  return true
})
```

无商品空态：

```tsx
{products.length === 0 && <div className="rounded-md bg-amber-50 p-3 text-sm text-amber-700">暂无可排期商品。请先将商品设为可排期，或等待当前竞拍结束。</div>}
```

提交按钮禁用条件增加：

```tsx
disabled={createLoading || createSubmitting || products.length === 0}
```

- [ ] **Step 6: 运行前端测试**

Run:

```bash
cd frontend/admin && npm test -- --run src/pages-new/__tests__/GoodsList.test.tsx src/pages-new/__tests__/AuctionList.createAuction.test.tsx
```

Expected: PASS。

- [ ] **Step 7: 提交**

```bash
git add frontend/admin/src/shared/api/types.ts frontend/admin/src/pages-new/GoodsList.tsx frontend/admin/src/pages-new/AuctionList.tsx frontend/admin/src/pages-new/__tests__/GoodsList.test.tsx frontend/admin/src/pages-new/__tests__/AuctionList.createAuction.test.tsx
git commit -m "feat(admin): align product and auction lifecycle UI"
```

---

### Task 9: 数据一致性检查与 test-service 约束固化

**Files:**
- Modify: `backend/test/scenario/antisnipe/factory.go`
- Create: `scripts/check-auction-product-consistency.sql`

- [ ] **Step 1: 修改 fixture 注释**

将 `backend/test/scenario/antisnipe/factory.go` 的 `Prepare` 注释改为：

```go
// Prepare 创建一个商品、竞拍规则和拍卖。
// 每次调用必须创建独立商品：auction-service 对同一 product_id 只允许一条 Pending/Ongoing/Delayed 活跃竞拍。
```

- [ ] **Step 2: 创建只读 SQL 检查命令**

创建 `scripts/check-auction-product-consistency.sql`：

```sql
-- 只读检查：auction.product_id 在 products 中缺失
SELECT a.id AS auction_id, a.product_id, a.creator_id
FROM auctions a
LEFT JOIN products p ON p.id = a.product_id
WHERE p.id IS NULL;

-- 只读检查：auction.creator_id 与 products.owner_id 不一致
SELECT a.id AS auction_id, a.product_id, a.creator_id, p.owner_id
FROM auctions a
JOIN products p ON p.id = a.product_id
WHERE a.creator_id IS NOT NULL
  AND p.owner_id IS NOT NULL
  AND a.creator_id <> p.owner_id;

-- 只读检查：同一商品存在多条活跃竞拍
SELECT product_id, COUNT(*) AS active_count
FROM auctions
WHERE status IN (0, 1, 2)
GROUP BY product_id
HAVING COUNT(*) > 1;
```

- [ ] **Step 3: 运行回归测试**

Run:

```bash
cd backend/test && go test ./... -run 'Test.*(AntiSnipe|Pressure|E2E)' -count=1
```

Expected: PASS。若仓库没有这些精确测试名，则运行：

```bash
cd backend/test && go test ./... -count=1
```

Expected: PASS。

- [ ] **Step 4: 提交**

```bash
git add backend/test/scenario/antisnipe/factory.go scripts/check-auction-product-consistency.sql
git commit -m "test: document one-active-auction fixture constraint"
```

---

## Final Verification

- [ ] Run product-service tests:

```bash
cd backend/product && go test ./...
```

Expected: PASS。

- [ ] Run auction-service tests:

```bash
cd backend/auction && go test ./...
```

Expected: PASS。

- [ ] Run admin frontend tests:

```bash
cd frontend/admin && npm test -- --run
```

Expected: PASS。

- [ ] Manual MySQL DDL check after booting auction-service:

```sql
SHOW COLUMNS FROM auctions LIKE 'active_product_key';
SHOW INDEX FROM auctions WHERE Key_name = 'uk_active_product';
```

Expected: `active_product_key` exists, and `uk_active_product` exists as a unique index.

- [ ] Manual API check through Gateway:

```bash
curl -sS -X POST http://localhost:8080/api/v1/auctions \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${MERCHANT_JWT}" \
  -d '{"product_id":1001,"duration":3600}'
```

Expected: `201`，响应 auction 包含 `product_id=1001`、`live_stream_id`、`status=0`。

---

## Self-Review

**Spec coverage:** §2.1 由 Task 1/8 覆盖；§2.3/§4.4 由 Task 4/5 覆盖；§2.4/§4.3 由 Task 2/3/6 覆盖；§3.1/§6.2 由 Task 7/8 覆盖；§3.2 由 Task 8 覆盖（已拍卖/流拍区分为前端客户端过滤，后端聚合 query 标注为已知限制，留待后续迭代）；§4.1/§4.2 由 Task 5/6/7 覆盖（直播间 active 校验含 Task 2 的禁用→409 用例）；§5.3/§7.4 由 Task 9 覆盖；§7.3 由 Task 9 SQL 覆盖。

**Placeholder scan:** 本计划没有未展开的占位指令。所有新增方法、请求体、核心测试和命令都给出具体内容。

**Type consistency:** `client.AuctionProductInfo`、`client.LiveStreamInfo`、`service.CreateAuctionRequest`、`client.ProductAuctionState` 在各任务中名称一致；`winner_id` 继续作为成交/流拍事实来源；`display_status` 仅为响应字段，不对应持久化列。身份/角色校验只在 handler 层（Task 6），service 层不定义/不返回身份类错误（已删除 `ErrCreateAuctionUnauthorized`/`ErrCreateAuctionForbidden` 避免不可达分支），`writeCreateAuctionError` 仅映射业务错误（400/409）。
