# Admin Order Management Backendized Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 补齐商家订单管理页面的后端搜索、真实状态统计和买家昵称头像展示。

**Architecture:** `product-service` 保持订单事实源，新增订单 admin 查询的 `search` 和 `summary`。买家昵称/头像通过 `auction-service` 已有 `/internal/users/batch` 内部 API 获取，避免跨库 JOIN 用户表。前端继续走 Gateway `/api/v1`。

**Tech Stack:** Go 1.24、Hertz、GORM、shopspring/decimal、React、TypeScript、Jest/Vitest。

---

## Files

- Modify: `backend/product/dao/order_admin.go`
- Modify: `backend/product/service/order_admin.go`
- Modify: `backend/product/handler/order_admin.go`
- Modify: `backend/product/handler/order_admin_test.go`
- Create: `backend/product/client/user_client.go`
- Create: `backend/product/client/user_client_test.go`
- Modify: `backend/product/main.go`
- Modify: `frontend/admin/src/shared/api/index.ts`
- Modify: `frontend/admin/src/pages-new/OrderList.tsx`
- Modify: `frontend/admin/src/pages-new/OrderDetail.tsx`
- Create or modify: `frontend/admin/src/pages-new/__tests__/OrderList.test.tsx`
- Create or modify: `frontend/admin/src/pages-new/__tests__/OrderDetail.test.tsx`

## Tasks

### Task 1: 后端订单搜索与状态摘要

**Files:**
- Test: `backend/product/handler/order_admin_test.go`
- Modify: `backend/product/dao/order_admin.go`
- Modify: `backend/product/service/order_admin.go`
- Modify: `backend/product/handler/order_admin.go`

- [ ] **Step 1: Write failing tests**

Add tests proving:

- `GET /api/v1/admin/orders?search=101` returns only order `101`.
- `GET /api/v1/admin/orders?search=玉镯` searches `products.name`.
- `GET /api/v1/admin/orders` returns `summary.pending_payment_count`, `summary.paid_count`, `summary.shipped_count`, `summary.completed_count`.
- merchant role summary is scoped to `seller_id = X-User-ID`.

- [ ] **Step 2: Verify red**

Run:

```bash
cd backend/product && go test ./handler -run 'TestOrderHandler_AdminList' -count=1
```

Expected: FAIL because `search` and `summary` are not implemented.

- [ ] **Step 3: Minimal implementation**

Add DAO/service methods that accept `search string` and return `OrderAdminSummary`. Search supports:

- numeric exact match on `orders.id`
- numeric exact match on `orders.winner_id`
- `products.name LIKE %search%`

Keep seller scope applied to both list and summary queries.

- [ ] **Step 4: Verify green**

Run:

```bash
cd backend/product && go test ./handler ./service ./dao -count=1
```

Expected: PASS.

### Task 2: 后端买家昵称头像补齐

**Files:**
- Create: `backend/product/client/user_client.go`
- Create: `backend/product/client/user_client_test.go`
- Modify: `backend/product/service/order_admin.go`
- Modify: `backend/product/handler/order_admin_test.go`
- Modify: `backend/product/main.go`

- [ ] **Step 1: Write failing tests**

Add tests proving:

- product user client calls `POST /internal/users/batch` with `X-Internal-Token`.
- `OrderService.ListAdminOrdersScoped` enriches `user_name` and `user_avatar` for matching `winner_id`.
- user client failure does not fail order list; buyer fields remain empty.

- [ ] **Step 2: Verify red**

Run:

```bash
cd backend/product && go test ./client ./service ./handler -run 'Test.*User|TestOrderHandler_AdminList' -count=1
```

Expected: FAIL because product client and enrichment do not exist.

- [ ] **Step 3: Minimal implementation**

Introduce a narrow `UserSummaryProvider` interface in `service/order_admin.go`:

```go
type UserSummaryProvider interface {
    BatchGetUserSummaries(ctx context.Context, ids []int64) (map[int64]UserSummary, error)
}
```

Implement HTTP client under `backend/product/client` using existing config values for auction service URL and internal token.

Inject the provider in `main.go` when constructing `OrderService`.

- [ ] **Step 4: Verify green**

Run:

```bash
cd backend/product && go test ./client ./service ./handler -count=1
```

Expected: PASS.

### Task 3: 前端订单列表真实化

**Files:**
- Modify: `frontend/admin/src/shared/api/index.ts`
- Modify: `frontend/admin/src/pages-new/OrderList.tsx`
- Create or modify: `frontend/admin/src/pages-new/__tests__/OrderList.test.tsx`

- [ ] **Step 1: Write failing tests**

Add tests proving:

- typing search term and submitting/refetching calls `orderApi.list({ search })`.
- status cards render real summary counts.
- buyer column shows `user_name` when present.
- Filter icon button is absent.
- More action exposes only `查看详情` or direct detail navigation.

- [ ] **Step 2: Verify red**

Run:

```bash
cd frontend/admin && npm test -- OrderList --runInBand
```

Expected: FAIL because summary/search buyer behavior is not wired.

- [ ] **Step 3: Minimal implementation**

Extend `orderApi.list` params with `search`; update response type with `summary`. Use debounced or explicit search trigger only if simple; otherwise call `fetchOrders` when search term changes after Enter/button submit. Remove no-op Filter button. Replace More icon no-op with a real “查看详情” action or remove it if row click is enough.

- [ ] **Step 4: Verify green**

Run:

```bash
cd frontend/admin && npm test -- OrderList --runInBand
```

Expected: PASS.

### Task 4: 前端订单详情买家展示

**Files:**
- Modify: `frontend/admin/src/pages-new/OrderDetail.tsx`
- Create or modify: `frontend/admin/src/pages-new/__tests__/OrderDetail.test.tsx`

- [ ] **Step 1: Write failing tests**

Add tests proving:

- detail page shows buyer `user_name` and avatar when provided.
- detail page falls back to `用户 #<user_id>` when `user_name` is empty.

- [ ] **Step 2: Verify red**

Run:

```bash
cd frontend/admin && npm test -- OrderDetail --runInBand
```

Expected: FAIL if tests require avatar/name behavior not yet implemented.

- [ ] **Step 3: Minimal implementation**

Render buyer avatar with existing UI primitives or plain image fallback. Do not add phone/address/物流 fields.

- [ ] **Step 4: Verify green**

Run:

```bash
cd frontend/admin && npm test -- OrderDetail --runInBand
```

Expected: PASS.

### Task 5: Full verification

Run:

```bash
cd backend/product && go test ./handler ./service ./dao ./client -count=1
cd frontend/admin && npm test -- --runInBand
cd frontend/admin && npm run build
```

Expected: all pass. Update SDD state with evidence and remaining risks.

