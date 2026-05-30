# H5 缺口闭环·子 spec D：OrderDetail 页面与 Home 未读数接入

**日期**：2026-05-30

**关联总览 spec**：[2026-05-30-h5-missing-interfaces-closure.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/superpowers/specs/2026-05-30-h5-missing-interfaces-closure.md)

**风格参考**：[2026-05-30-user-touchpoints-backend-design-adapted.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/superpowers/specs/2026-05-30-user-touchpoints-backend-design-adapted.md)

**适用仓库**：`/Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc`

---

## 1. 范围

本子 spec 仅覆盖以下两项纯前端能力，**不修改任何后端代码**：

| 编号 | 能力 | 优先级 |
|---|---|---|
| F-D1 | H5 新增 OrderDetail 页面，路由 `/order/:id`，复用既有 `GET /api/v1/orders/:id`，仅展示状态，不实现支付链路 | P2 |
| F-D2 | Home 顶部通知图标接入真实未读数，复用既有 `GET /api/v1/notifications/unread-count`，无轮询 | P2 |

明确不在范围内的事项：

- 后端 handler、model、路由、迁移脚本一律不动。
- 不实现订单支付按钮（支付链路本期不做）。
- 不实现订单状态实时同步（不接 WebSocket）。
- 不实现轮询刷新未读数。

---

## 2. 真实代码事实

| 事实 | 位置 |
|---|---|
| 订单详情后端已存在 | [order.go#L73-L94](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/handler/order.go) |
| Order model 含 `Status`/`PaidAt`/`ShippedAt`/`CompletedAt` | [order.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/model/order.go) |
| `OrderStatus`：0 待支付 / 1 已支付 / 2 已发货 / 3 已完成 | [order.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/model/order.go) |
| 前端 `orderApi.get(id)` 已存在 | [api.ts#L348-L349](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts) |
| 前端路由配置（无 `/order/:id`） | [App.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/App.tsx) |
| 前端 `services/api.ts` 中**无 `notificationApi` 命名空间**，需新增 | [api.ts](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts) |
| `BadgeDot` 组件已实现，已用于 BottomNav 与个人中心 | [BadgeDot](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/components/BadgeDot) |
| Home 顶部通知图标 | [Home/index.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Home/index.tsx) |
| 通知列表中含 `data.order_id` 时需跳转订单详情 | [Notifications/index.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Notifications/index.tsx) |
| 中标结果页跳订单详情入口 | [Auction/Result.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Auction/Result.tsx) |
| OrderDetail 页面**不存在** | [pages/](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages) |

---

## 3. 共享约定

- 所有 HTTP 经 `gateway-service`，业务码遵循 `{0, 200}` 视为成功。
- OrderDetail 页面支持深色/浅色主题：CSS Modules + `--touchpoint-*` 主题变量。
- 可点击控件高度 ≥ 44px。
- OrderDetail 数据刷新策略：进入页面 fetch 一次 + 下拉刷新；**不接 WebSocket，不轮询**。
- Home 未读数刷新策略：组件 mount 时调用一次 + `visibilitychange` 切回时再调用一次；**不轮询**。
- 401 由全局 axios 拦截器处理，本 spec 不重复实现。

---

## 4. F-D1：OrderDetail 页面规格

### 4.1 路由

- 路径：`/order/:id`
- 在 [App.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/App.tsx) 注册，包裹 `PrivateRoute`，未登录跳 `/login`。
- `:id` 必须为正整数；非法 id 路由层不拦截，进入页面后由数据层报 404。

### 4.2 数据契约

- 数据来源：`orderApi.get(Number(id))`，对应后端 [GET /api/v1/orders/:id](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/handler/order.go)。
- 响应字段以 [Order model](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/model/order.go) 为准：`id`/`auction_id`/`product_id`/`winner_id`/`final_price`/`status`/`paid_at?`/`shipped_at?`/`completed_at?`/`created_at`/`updated_at`。
- 商品摘要展示：当前 Order 模型不含商品冗余字段；本期仅展示 `product_id` 与 `auction_id` 链接占位，**不额外调用 product 详情接口**（避免越界，留待后续 spec）。

### 4.3 UI 区块（自上而下）

1. 顶部导航：返回按钮 + 标题「订单详情」。
2. 状态徽章：根据 `status` 渲染
   - `0` → 「待支付」（warning 色）
   - `1` → 「已支付」（info 色）
   - `2` → 「已发货」（success 色）
   - `3` → 「已完成」（neutral 色）
3. 商品摘要卡片：展示 `auction_id`、`product_id`、`final_price`（人民币格式）。
4. 订单金额：`final_price` 主显，单位 ¥。
5. 时间线：`created_at`（创建）→ `paid_at`（支付，未支付时显示「—」）→ `shipped_at`（发货，未发货时显示「—」）。已完成时间 `completed_at` 作为终点节点。
6. 操作区：
   - 「返回」按钮（始终可见）。
   - 「联系客服」按钮（占位，点击仅触发 toast「客服功能即将上线」）。
   - **不渲染支付按钮**。

### 4.4 状态机

| 场景 | UI |
|---|---|
| loading | 骨架屏或 LoadingSpinner |
| 成功 | 渲染 4.3 区块 |
| 后端 404（订单不存在） | 居中文案「订单不存在」+「返回」按钮 |
| 网络错误 | 居中错误态 +「重试」按钮 |

### 4.5 文件结构

- 新增：[pages/Order/Detail.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Order/Detail.tsx)
- 新增：[pages/Order/Detail.module.css](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Order/Detail.module.css)
- 新增：[pages/Order/__tests__/Detail.test.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Order/__tests__/Detail.test.tsx)
- 在 [App.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/App.tsx) 中 `lazy` 导入并注册路由。

---

## 5. F-D2：Home 未读数接入规格

### 5.1 API 客户端补充

在 [services/api.ts](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts) 新增 `notificationApi` 命名空间（当前文件无该命名空间），至少包含：

- `getUnreadCount(): Promise<{ unread_count: number }>` → `GET /notifications/unread-count`

返回值结构以网关响应为准；前端只消费 `unread_count` 字段。

### 5.2 Home 集成

在 [Home/index.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Home/index.tsx) 顶部通知图标处：

- 组件 mount 时调用 `notificationApi.getUnreadCount()`。
- 监听 `document.visibilitychange`，从隐藏切回可见时再调用一次；卸载时移除监听器。
- `unread_count > 0` 时在通知图标右上角渲染 [BadgeDot](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/components/BadgeDot) 组件，传递 count（沿用 BadgeDot 现有展示约定，例如 99+ 截断由组件自身负责）。
- **不在 Home 启用任何 setInterval / WebSocket / SSE 轮询机制**。

### 5.3 失败降级

- 接口失败（含网络异常、5xx、业务码非成功）：UI 不显示红点，不弹错误 toast，不阻塞 Home 渲染。
- 失败仅 `console.warn`，不计入埋点。

### 5.4 鉴权

- 未登录用户进入 Home 时不发起 `getUnreadCount` 请求（结合 `useAuth().isAuthenticated` 判断），避免 401 噪声。

---

## 6. 跳转入口集成

需要跳转到 `/order/:id` 的来源：

| 来源 | 触发条件 | 行为 |
|---|---|---|
| [Notifications/index.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Notifications/index.tsx) 通知卡片 | 通知 `data.order_id` 存在 | `navigate('/order/' + order_id)` |
| [Auction/Result.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Auction/Result.tsx) 中标后按钮 | 仅当 result 接口已返回 `order_id` | 启用「查看订单」按钮，点击 `navigate('/order/' + order_id)` |
| 同上，缺失 `order_id` | 兜底 | 按钮**置灰禁用**并显示文案「订单生成中」，**不创建死链** |

补充：result 接口返回 `order_id` 字段属子 spec C 范围；本 spec 仅承接已存在字段时的跳转行为，不修改 result 接口契约。

---

## 7. 测试要求

### 7.1 新增测试

- [pages/Order/__tests__/Detail.test.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Order/__tests__/Detail.test.tsx) 必须覆盖：
  - loading 态渲染。
  - 成功态：`status` 各取值（0/1/2/3）的状态徽章与时间线节点渲染。
  - 后端返回 404：渲染「订单不存在」+ 返回按钮。
  - 操作区：「联系客服」点击触发 toast，无支付按钮 DOM。

### 7.2 api.ts 单测扩展

- 在现有 api 单测套件中新增 `notificationApi.getUnreadCount` 用例：
  - 正常返回 `{ unread_count: N }`。
  - HTTP 失败时 promise reject。

### 7.3 受影响的现有测试

- [Home.integration.test.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/__tests__/integration/Home.integration.test.tsx)：mock `notificationApi.getUnreadCount`，覆盖：
  - `unread_count = 0` → 无 BadgeDot。
  - `unread_count > 0` → 顶部通知图标显示 BadgeDot。
  - 接口失败 → 无 BadgeDot 且 Home 正常渲染。

---

## 8. 风险与边界

| 项 | 决策 | 理由 |
|---|---|---|
| 不实现支付按钮 | 本期支付链路不做，OrderDetail 仅展示 | 范围控制，避免半成品 |
| 不接 WebSocket 实时同步订单状态 | 进入页面 fetch + 下拉刷新 | 订单状态对实时性不敏感，节约连接成本 |
| Home 未读数不轮询 | mount + visibilitychange | 避免性能开销，足以覆盖主要使用场景 |
| result 缺 `order_id` 时禁用按钮 | 不创建死链 | `/order/undefined` 会触发 404，UX 更差 |
| 商品摘要不调用 product 详情 | 仅展示 `product_id`/`auction_id` | 当前 Order model 无冗余字段，新增调用越出本 spec 范围 |
| 401 由全局拦截器处理 | 本 spec 不重复 | 与现有 axios 实例契约保持一致 |
