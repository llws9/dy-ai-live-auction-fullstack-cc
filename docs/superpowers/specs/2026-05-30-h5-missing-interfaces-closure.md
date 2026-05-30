# H5 接口缺口闭环 · 总览 Spec

**日期**：2026-05-30

**输入文档**：[missing-interfaces.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/docs/mobile-ui-migration/missing-interfaces.md)

**适用仓库**：`/Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc`

**关联子 Spec**：

- [子 spec A · 用户中心数据闭环](./2026-05-30-h5-missing-a-user-center.md)
- [子 spec B · 直播间详情 + 关注语义重命名](./2026-05-30-h5-missing-b-livestream.md)
- [子 spec C · 商品/竞拍/分类数据契约](./2026-05-30-h5-missing-c-product-auction.md)
- [子 spec D · OrderDetail 页面 + Home 未读数接入](./2026-05-30-h5-missing-d-order-detail.md)
- [总览 Tasks](./2026-05-30-h5-missing-tasks.md)

---

## 1. 范围与排除项

### 1.1 用户已确认的边界

| 维度 | 决策 |
|---|---|
| 范围排除 | 订单**支付链路**（Gateway/Product `/orders/:id/pay` 方法不一致）不修；直播**聊天**协议不做 |
| 钱包 | 仅做 `GET /api/v1/user/balance` 余额展示；**不做**充值、保证金、资金流 |
| 收藏对象 | 复用现有 `live_stream` follow 数据源；**仅 UI 文案/图标重命名**为「收藏」，**不新建** `favorites` 表 |
| Following 页面 | 保留路由 `/follow`，重命名为「我的收藏」，复用同一数据源 |
| 输出形态 | 单一总 spec + 4 个领域子 spec；先完成全部 spec 再统一进入 execution |

### 1.2 取消的子需求

原 missing-interfaces.md 中以下条目**本期不做**：

- 订单支付方法统一（属支付链路）
- 直播聊天 WS 协议、`/live-streams/:id/messages` HTTP 接口
- 商品分享后端记录（`POST /share-events`），仅保留浏览器原生 Web Share 选项
- 钱包充值、保证金接口与资金流模型

### 1.3 与 missing-interfaces.md 的事实校正

仓库当前状态已与原文档发生偏离，本次 spec 基于以下**校正事实**而非原文档：

| 原文档判定 | 真实状态 | 来源 |
|---|---|---|
| Gateway 未暴露 `/categories` | **已暴露**（`v1.GET("/categories", ...)`） | [router.go#L126](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/router/router.go#L126) |
| 收藏需要新建 `favorites` 表/接口 | 用户决策：仅前端文案重命名，复用 follow | 用户 Q&A |
| 钱包/保证金需要补全 | 用户决策：仅做 `/user/balance` 只读展示 | 用户 Q&A |
| 订单支付方法统一 | 用户决策：本期不做 | 用户 Q&A |

### 1.4 真正落地的能力清单

| 编号 | 能力 | 子 Spec | 优先级 |
|---|---|---|---|
| F-A1 | `GET /api/v1/users/me/stats` 用户统计 | A | P1 |
| F-A2 | `GET /api/v1/user/balance` 余额（只读） | A | P2 |
| F-A3 | `/api/v1/users/me/addresses` 收货地址 CRUD | A | P2 |
| F-B1 | `GET /api/v1/live-streams/:id` 字段扩展（host/viewer/video/is_following） | B | P1 |
| F-B2 | `GET /api/v1/live-streams/:id/follow-status` | B | P1 |
| F-B3 | Following 列表卡片字段补齐（host_avatar/viewer_count/auction_count） | B | P2 |
| F-B4 | UI 重命名：FollowButton → CollectButton；Following → MyCollections | B | P1 |
| F-C1 | `GET /api/v1/auctions` 增加 `category_id` 过滤 + 内嵌 `product` 摘要 | C | P1 |
| F-C2 | `GET /api/v1/auctions/:id/result` 扩展 `won_bid` + `product` 摘要 | C | P1 |
| F-C3 | `GET /api/v1/orders/history` 改为 JWT 用户语义，扩展 `product/my_highest_bid/ended_at` | C | P0 |
| F-C4 | 规则归属语义修正：明确 `/products/:id/rules` 中 `product_id` 不再混用 `auction_id` | C | P2 |
| F-D1 | H5 `/order/:id` OrderDetail 页面 | D | P2 |
| F-D2 | Home 接入真实 `notification.unread_count` | D | P2 |

---

## 2. 第一性原则

### 2.1 SSOT（Single Source of Truth）

- 不为「收藏」单独建表，避免与现有 `user_live_stream_follows` 形成双重事实源。
- result、history、live-stream 详情的数据全部回到各自服务的领域模型，不在 Gateway 拼数据除非有明确 BFF 收益。
- 用户身份一律从 JWT（Gateway `middleware.JWTAuth` 注入 `user_id`）读取，禁止 query `user_id` 参数。

### 2.2 数据驱动 UI

- UI 状态完全从 API 派生：`is_following` 由后端权威提供，不再使用前端 `initialFollowed` props 兜底默认 `false`。
- 缺字段时显示空态/禁用，不使用本地 mock 或硬编码降级数据。

### 2.3 契约前置

- 所有新增/修改接口先在 spec 中明确请求路径、方法、鉴权要求、Request 字段、Response 形态、错误语义。
- Gateway 路由必须在 [router.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/router/router.go) 同步注册；前端不得绕过 Gateway 直连下游。

### 2.4 接口分类不变更

继续遵守现有约束：

- 所有 H5 流量经 `gateway-service`（端口 8080）。
- 业务成功码统一 `{0, 200}`，由 [api.ts#L12](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts#L12) 校验。
- 401 触发 token 清理 + 登录跳转。

---

## 3. 跨领域共享约定

### 3.1 统一分页响应

所有列表型接口响应**必须**为：

```json
{
  "code": 200,
  "data": {
    "items": [...],
    "total": 0,
    "page": 1,
    "page_size": 20
  }
}
```

不再使用 `data.list`。前端 adapter 暂时兼容 `items ?? list ?? data.items ?? data.list`，但本期新增/修改的接口必须只产出 `items`。

### 3.2 用户身份注入

- Gateway `middleware.JWTAuth` 将 `user_id` 写入 `app.RequestContext`。
- 下游服务通过 `c.GetInt64("user_id")` 读取。
- 所有 `/users/me/*`、`/orders/history`、`/user/balance` 等用户域接口禁止接受 `user_id` query 参数。

### 3.3 关注/收藏语义双名映射（一期）

- 后端：保持 `follow`、`unfollow`、`followed-live-streams` 命名不变。
- 前端：组件、文案、图标统一为「收藏」。
- 下一期可考虑后端重命名（DB 表 `user_live_stream_follows` → `user_live_stream_collections`），本期**不动**。

### 3.4 错误响应结构

```json
{ "code": 400, "message": "..." }
```

`message` 用于用户可见提示。下游服务 5xx 时 Gateway 应透传 5xx 而非降级为 200。

---

## 4. 优先级与里程碑

| 优先级 | 内容 | 原因 |
|---|---|---|
| P0 | F-C3 `orders/history` JWT 化 | 安全：当前任意用户可查任意 user_id 历史 |
| P1 | F-A1, F-B1, F-B2, F-B4, F-C1, F-C2 | 核心数据闭环（首页分类、直播间真实信息、收藏交互） |
| P2 | F-A2, F-A3, F-B3, F-C4, F-D1, F-D2 | Profile 完善、Following 卡片精修、订单详情承接 |

里程碑：

- M1（P0）：`orders/history` JWT 化，含单测
- M2（P1）：用户 stats、live-stream 详情扩展、follow-status、auctions 分类过滤、result 扩展、UI 重命名
- M3（P2）：余额/地址/Following 卡片/规则归属/OrderDetail/Home 未读数

---

## 5. 测试与验收

### 5.1 后端

- 每个新增/修改 handler 必须配套单元测试，覆盖鉴权、参数校验、空数据、正常响应。
- 涉及 JWT 注入的接口必须验证「带 token / 无 token / token 无效」三态。
- 新增字段必须有迁移脚本或 `AutoMigrate` 验证。

### 5.2 前端

- 现有受影响测试（[FollowButton.test.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/components/__tests__/FollowButton.test.tsx)、[Following.test.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Follow/__tests__/Following.test.tsx)、[Profile.test.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/User/__tests__/Profile.test.tsx)、[LiveRoom.test.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/__tests__/LiveRoom.test.tsx)）必须随重命名/契约调整同步更新。
- adapter 函数必须有单测，验证「字段缺失 / 字段齐全 / 旧字段名」三态。

### 5.3 集成

- Gateway 路由变更后必须重启 `gateway-service` 并验证 `/health`。
- E2E 用例 [e2e/auction.spec.ts](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/e2e/auction.spec.ts) 需在影响路径上回归。

---

## 6. 不在本期处理的已知遗留

- 订单支付方法统一（POST vs PUT）
- 聊天协议、分享后端
- 钱包充值/保证金/资金流
- 后端 follow 命名重构为 collection
- 通知 `data` 字段拉平为强类型 schema（保持现状 JSON）
