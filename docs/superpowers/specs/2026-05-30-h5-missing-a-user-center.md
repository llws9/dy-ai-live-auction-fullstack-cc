# 子 Spec A · 用户中心数据闭环

**日期**：2026-05-30

**总览 Spec**：[2026-05-30-h5-missing-interfaces-closure.md](./2026-05-30-h5-missing-interfaces-closure.md)

**关联子 Spec**：

- [子 spec B · 直播间详情 + 关注语义重命名](./2026-05-30-h5-missing-b-livestream.md)
- [子 spec C · 商品/竞拍/分类数据契约](./2026-05-30-h5-missing-c-product-auction.md)
- [子 spec D · OrderDetail 页面 + Home 未读数接入](./2026-05-30-h5-missing-d-order-detail.md)

**前端入口**：[Index.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/User/Index.tsx)

---

## 1. 范围

### 1.1 本子 spec 落地的能力

| 编号 | 能力 | 优先级 |
|---|---|---|
| F-A1 | `GET /api/v1/users/me/stats` 用户统计：`following_count` / `auction_history_count` / `won_count` | P1 |
| F-A2 | `GET /api/v1/user/balance` 余额（只读、仅展示） | P2 |
| F-A3 | `/api/v1/users/me/addresses` 收货地址 CRUD + 设默认 | P2 |

### 1.2 明确不做

- 充值、提现、保证金、冻结/解冻等任何资金流写入路径。
- 交易/订单与 `user_balance` 之间的联动。
- 收藏建模（沿用 `user_live_stream_follows`，`following_count` 即收藏数，详见总览 spec）。
- 地址软删除、地址校验/补全（行政区代码、邮编等）。
- 多币种结算逻辑；`currency` 字段为后续扩展占位，本期固定 `CNY`。

---

## 2. 接口契约

所有接口统一满足：

- 路径前缀：`/api/v1`，必经 `gateway-service`。
- 鉴权：`authGroup` 下 JWT 必填；`user_id` 由 [middleware.JWTAuth](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/router/router.go#L50) 注入，**禁止** query/body 传 `user_id`。
- 成功响应：`{"code": 200, "data": {...}}`；列表使用 `items / total / page / page_size`。
- 错误响应：`{"code": <http_status>, "message": "..."}`。

### 2.1 F-A1 用户统计

| 项 | 内容 |
|---|---|
| Method/Path | `GET /api/v1/users/me/stats` |
| 鉴权 | JWT |
| Request | 无 query 参数 |

**Response（成功）**：

```json
{
  "code": 200,
  "data": {
    "following_count": 12,
    "auction_history_count": 34,
    "won_count": 5
  }
}
```

字段语义：

| 字段 | 来源 | 计算口径 |
|---|---|---|
| `following_count` | `auction-service.user_live_stream_follows` | `WHERE user_id = ? AND status = active` 计数 |
| `auction_history_count` | `product-service.orders` | `WHERE user_id = ?` 计数（与 `/orders/history` 一致） |
| `won_count` | `product-service.orders` | `WHERE user_id = ? AND is_winner = 1` 计数 |

**错误码**：`401`（未登录）、`500`（聚合失败，见 §9）。

**前端调用入口建议**：在 [api.ts](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts#L282) 的 `userApi` 中新增 `getStats: () => get<any>('/users/me/stats')`。

### 2.2 F-A2 余额（只读）

| 项 | 内容 |
|---|---|
| Method/Path | `GET /api/v1/user/balance` |
| 鉴权 | JWT |
| Request | 无 |

**Response（成功）**：

```json
{
  "code": 200,
  "data": {
    "available_amount": "0.00",
    "frozen_amount": "0.00",
    "currency": "CNY",
    "updated_at": "2026-05-30T10:00:00+08:00"
  }
}
```

约定：

- `user_balance` 表无记录时返回上述零值结构（**不返回 404**），`updated_at` 取当前服务器时间。
- 金额一律字符串（`decimal` 序列化），前端使用 `toNumber` 转换。
- 不暴露任何写接口；新增/修改余额由后续资金链路 spec 处理。

**错误码**：`401`、`500`。

**前端调用入口建议**：保留 [api.ts](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts#L290) `userApi.getBalance`，对接新契约后调整 `BalanceData` 类型字段为 `available_amount / frozen_amount / currency`。

### 2.3 F-A3 收货地址 CRUD

| 编号 | Method/Path | 鉴权 | 用途 |
|---|---|---|---|
| A3.1 | `GET /api/v1/users/me/addresses` | JWT | 列表（不分页，最多 20 条，见 §9） |
| A3.2 | `POST /api/v1/users/me/addresses` | JWT | 创建 |
| A3.3 | `PUT /api/v1/users/me/addresses/:id` | JWT | 更新 |
| A3.4 | `DELETE /api/v1/users/me/addresses/:id` | JWT | 删除（硬删除） |
| A3.5 | `POST /api/v1/users/me/addresses/:id/default` | JWT | 设为默认 |

**列表响应**：

```json
{
  "code": 200,
  "data": {
    "items": [
      {
        "id": 1001,
        "recipient_name": "张三",
        "phone": "138****0000",
        "province": "北京市",
        "city": "北京市",
        "district": "海淀区",
        "detail": "中关村大街 1 号",
        "is_default": true,
        "created_at": "2026-05-30T10:00:00+08:00",
        "updated_at": "2026-05-30T10:00:00+08:00"
      }
    ],
    "total": 1
  }
}
```

**创建 / 更新请求体**（字段语义一致；更新允许部分字段缺省，按 PATCH 语义合并）：

```json
{
  "recipient_name": "张三",
  "phone": "13800000000",
  "province": "北京市",
  "city": "北京市",
  "district": "海淀区",
  "detail": "中关村大街 1 号",
  "is_default": false
}
```

**字段约束**：

| 字段 | 约束 |
|---|---|
| `recipient_name` | 必填，1–32 字符 |
| `phone` | 必填，11 位中国手机号正则 |
| `province` / `city` / `district` | 必填，各 1–32 字符 |
| `detail` | 必填，1–128 字符 |
| `is_default` | 可选，默认 `false`；当用户当前无地址时，首次创建强制为 `true` |

**默认地址语义**：

- 同一 `user_id` 至多一条 `is_default = true`；`POST /:id/default` 与创建/更新中的 `is_default=true` 须在事务中将其它行置为 `false`。
- 删除当前默认地址后**不**自动选举新默认；前端在用户进入下单流程时需提示选择。

**错误码**：

| Code | 场景 |
|---|---|
| 400 | 字段校验失败、地址数量超过 20 条 |
| 401 | 未登录 |
| 403 | 操作非本人地址 |
| 404 | `:id` 不存在或不属于当前 `user_id` |

**前端调用入口建议**：在 [api.ts](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts) 中新增 `addressApi`（list / create / update / remove / setDefault）。

---

## 3. 数据模型

### 3.1 `user_balance`（归属：`auction-service`）

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| `user_id` | `bigint` | PRIMARY KEY | 与 `users.id` 一致 |
| `available_amount` | `decimal(18,2)` | NOT NULL DEFAULT 0 | 可用余额 |
| `frozen_amount` | `decimal(18,2)` | NOT NULL DEFAULT 0 | 冻结余额（本期恒为 0） |
| `currency` | `varchar(8)` | NOT NULL DEFAULT 'CNY' | 币种 |
| `updated_at` | `datetime` | NOT NULL | 更新时间 |

**DDL 草案**：

```sql
CREATE TABLE IF NOT EXISTS user_balance (
  user_id          BIGINT       NOT NULL PRIMARY KEY,
  available_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
  frozen_amount    DECIMAL(18,2) NOT NULL DEFAULT 0,
  currency         VARCHAR(8)    NOT NULL DEFAULT 'CNY',
  updated_at       DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

归属理由：用户域当前位于 `auction-service`（[user.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/model/user.go)），余额读路径与 `users/me` 同服务最短。

### 3.2 `user_addresses`（归属：`auction-service`）

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| `id` | `bigint` | PK, AUTO_INCREMENT | |
| `user_id` | `bigint` | NOT NULL, INDEX | 拥有者 |
| `recipient_name` | `varchar(32)` | NOT NULL | 收件人 |
| `phone` | `varchar(20)` | NOT NULL | 联系电话 |
| `province` | `varchar(32)` | NOT NULL | 省 |
| `city` | `varchar(32)` | NOT NULL | 市 |
| `district` | `varchar(32)` | NOT NULL | 区 |
| `detail` | `varchar(128)` | NOT NULL | 详细地址 |
| `is_default` | `tinyint(1)` | NOT NULL DEFAULT 0 | 是否默认 |
| `created_at` | `datetime` | NOT NULL | |
| `updated_at` | `datetime` | NOT NULL | |

索引：`KEY idx_user (user_id)`；可选 `KEY idx_user_default (user_id, is_default)` 用于默认地址快速定位。

**DDL 草案**：

```sql
CREATE TABLE IF NOT EXISTS user_addresses (
  id             BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id        BIGINT       NOT NULL,
  recipient_name VARCHAR(32)  NOT NULL,
  phone          VARCHAR(20)  NOT NULL,
  province       VARCHAR(32)  NOT NULL,
  city           VARCHAR(32)  NOT NULL,
  district       VARCHAR(32)  NOT NULL,
  detail         VARCHAR(128) NOT NULL,
  is_default     TINYINT(1)   NOT NULL DEFAULT 0,
  created_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_user (user_id),
  KEY idx_user_default (user_id, is_default)
);
```

归属理由：地址簿本质属于用户域，归 `auction-service` 与 `users.id` 同库同服务，避免跨服务事务。

---

## 4. 实现位置

### 4.1 `auction-service`

| 关注点 | 路径 |
|---|---|
| Balance Model | `backend/auction/model/user_balance.go` |
| Balance DAO | `backend/auction/dao/user_balance.go` |
| Balance Service | `backend/auction/service/user_balance.go` |
| Balance Handler | 复用 `backend/auction/handler/user.go` 新增方法或新建 `user_balance.go` |
| Address Model | `backend/au