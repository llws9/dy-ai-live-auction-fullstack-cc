# 商家订单管理后端化设计

## 背景

当前管理端“订单管理”页面已接入订单列表、订单详情和商家发货接口，但仍存在几个未完成链路：

- 顶部待支付/待发货统计不是真实后端数据。
- 搜索框只过滤当前页，不能服务端检索订单。
- 筛选图标和更多操作是空交互。
- 买家展示只能 fallback 到 `用户 #id`。

本次后端已具备竞拍结束创建订单链路，但项目暂不建设支付、物流、导出和催付能力。

## 目标

将商家订单管理页面补齐为“真实订单运营入口”：服务端搜索、真实状态统计、可解释的操作区，以及买家昵称/头像展示。

## 范围

### 本次实现

- `GET /api/v1/admin/orders` 增加 `search` 查询参数。
- `GET /api/v1/admin/orders` 返回状态计数摘要。
- `product-service` 通过内部 API 调用 `auction-service` 的 `POST /internal/users/batch`，批量补齐订单买家 `username/avatar`。
- 管理端订单列表使用后端搜索和真实状态计数。
- 管理端详情页展示买家昵称、头像和用户 ID。
- 更多操作只保留真实可用动作：查看详情。
- 空 `Filter` 按钮不再展示为可点击无效功能。

### 明确不做

- 导出订单。
- 催付。
- 支付链路。
- 物流链路。
- 真实佣金、运费和结算。
- 买家手机号、收货地址等隐私字段。

## 架构

前端所有请求继续走 Gateway `/api/v1`，不直连后端子服务。

订单数据所有权仍在 `product-service`：

- `product-service` 查询 `orders` 和 `products`。
- `product-service` 不跨库 JOIN 用户表。
- 买家基础资料由 `auction-service` 持有，通过已有内部接口 `POST /internal/users/batch` 获取。

Gateway 继续负责外部鉴权和角色约束：

- 商家可读自己的订单。
- 平台管理员可读全量订单。
- 商家范围由 `product-service` 基于 `X-User-ID` 和 `seller_id` 再做防御性约束。

## API 契约

### `GET /api/v1/admin/orders`

新增 query：

- `search`：可选。支持订单 ID、商品名称、买家用户 ID。

响应 `data` 新增：

```json
{
  "list": [],
  "total": 0,
  "page": 1,
  "page_size": 20,
  "summary": {
    "pending_payment_count": 0,
    "paid_count": 0,
    "shipped_count": 0,
    "completed_count": 0
  }
}
```

订单项新增可选字段：

```json
{
  "user_name": "张三",
  "user_avatar": "https://..."
}
```

用户服务不可用时，订单接口不失败；买家信息降级为空，前端 fallback 为 `用户 #id`。订单列表属于核心能力，买家昵称/头像是增强展示。

## 测试策略

- Product handler/DAO/service：先写失败测试覆盖 `search`、`summary` 和买家摘要补齐。
- Product client：先写失败测试覆盖内部用户批量接口请求、鉴权头和响应解析。
- Admin 前端：先写失败测试覆盖搜索走 API、统计卡片显示真实计数、买家昵称显示、无效 Filter 移除。
- 验证命令：
  - `cd backend/product && go test ./handler ./service ./dao ./client -count=1`
  - `cd frontend/admin && npm test -- --runInBand`
  - `cd frontend/admin && npm run build`

