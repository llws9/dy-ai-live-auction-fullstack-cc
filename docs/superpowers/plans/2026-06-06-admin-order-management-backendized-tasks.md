# 商家订单管理后端化任务清单

- [ ] T001: 后端订单搜索与状态摘要
  - 范围：`backend/product/dao/order_admin.go`、`backend/product/service/order_admin.go`、`backend/product/handler/order_admin.go`、`backend/product/handler/order_admin_test.go`
  - TDD：先补 handler/DAO/service 失败测试，再实现 `search` 和 `summary`。
  - 验证：`cd backend/product && go test ./handler ./service ./dao -count=1`

- [ ] T002: 后端买家昵称头像补齐
  - 范围：`backend/product/client/user_client.go`、`backend/product/client/user_client_test.go`、`backend/product/service/order_admin.go`、`backend/product/main.go`
  - TDD：先补内部用户批量 client 测试和订单 enrichment 测试，再实现 HTTP client 与 service 注入。
  - 验证：`cd backend/product && go test ./client ./service ./handler -count=1`

- [ ] T003: 前端订单列表真实化
  - 范围：`frontend/admin/src/shared/api/index.ts`、`frontend/admin/src/pages-new/OrderList.tsx`、`frontend/admin/src/pages-new/__tests__/OrderList.test.tsx`
  - TDD：先补搜索 API 参数、真实 summary 卡片、买家昵称展示和无效 Filter 移除测试，再实现页面。
  - 验证：`cd frontend/admin && npm test -- OrderList --runInBand`

- [ ] T004: 前端订单详情买家展示
  - 范围：`frontend/admin/src/pages-new/OrderDetail.tsx`、`frontend/admin/src/pages-new/__tests__/OrderDetail.test.tsx`
  - TDD：先补昵称/头像展示与 fallback 测试，再实现详情页展示。
  - 验证：`cd frontend/admin && npm test -- OrderDetail --runInBand`

- [ ] T005: 全量验证与状态归档
  - 范围：SDD 状态文件、必要文档更新。
  - 验证：
    - `cd backend/product && go test ./handler ./service ./dao ./client -count=1`
    - `cd frontend/admin && npm test -- --runInBand`
    - `cd frontend/admin && npm run build`

