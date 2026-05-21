# API 接口文档

## 基础信息

- **Base URL**: `http://localhost:8080/api/v1`
- **认证方式**: `X-User-ID` Header（模拟认证）
- **数据格式**: JSON
- **字符编码**: UTF-8

---

## 1. 商品服务 (Product Service)

### 1.1 创建商品

**接口**: `POST /products`

**请求头**:
```
Content-Type: application/json
```

**请求体**:
```json
{
  "name": "商品名称",
  "description": "商品描述",
  "images": ["https://example.com/image1.jpg", "https://example.com/image2.jpg"]
}
```

**参数说明**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 商品名称，最大128字符 |
| description | string | 否 | 商品描述 |
| images | array | 否 | 商品图片URL列表 |

**响应示例**:
```json
{
  "id": 1,
  "name": "商品名称",
  "description": "商品描述",
  "images": ["https://example.com/image1.jpg"],
  "status": 0,
  "created_at": "2026-05-21T18:00:00+08:00"
}
```

**状态码**:
- `201`: 创建成功
- `400`: 参数错误
- `500`: 服务器错误

---

### 1.2 获取商品列表

**接口**: `GET /products`

**查询参数**:
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| status | int | 否 | - | 商品状态（0=草稿, 1=已发布）|
| page | int | 否 | 1 | 页码 |
| page_size | int | 否 | 20 | 每页数量（最大100）|

**响应示例**:
```json
{
  "items": [
    {
      "id": 1,
      "name": "商品名称",
      "description": "商品描述",
      "images": ["https://example.com/image1.jpg"],
      "status": 1,
      "created_at": "2026-05-21T18:00:00+08:00"
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1
}
```

**状态码**:
- `200`: 成功

---

### 1.3 获取商品详情

**接口**: `GET /products/:id`

**路径参数**:
| 参数 | 类型 | 说明 |
|------|------|------|
| id | int | 商品ID |

**响应示例**:
```json
{
  "id": 1,
  "name": "商品名称",
  "description": "商品描述",
  "images": ["https://example.com/image1.jpg"],
  "status": 1,
  "created_at": "2026-05-21T18:00:00+08:00"
}
```

**状态码**:
- `200`: 成功
- `404`: 商品不存在

---

### 1.4 更新商品

**接口**: `PUT /products/:id`

**请求体**:
```json
{
  "name": "新商品名称",
  "description": "新商品描述",
  "images": ["https://example.com/new-image.jpg"]
}
```

**状态码**:
- `200`: 更新成功
- `404`: 商品不存在

---

### 1.5 删除商品

**接口**: `DELETE /products/:id`

**状态码**:
- `200`: 删除成功
- `404`: 商品不存在

---

### 1.6 配置竞拍规则

**接口**: `POST /products/:id/rules`

**请求体**:
```json
{
  "start_price": 0,
  "increment": 10,
  "cap_price": 1000,
  "duration": 300,
  "delay_duration": 30,
  "max_delay_time": 180,
  "trigger_delay_before": 30
}
```

**参数说明**:
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| start_price | float | 否 | 0 | 起拍价（元）|
| increment | float | 是 | - | 加价幅度（元）|
| cap_price | float | 否 | - | 封顶价（元）|
| duration | int | 是 | - | 竞拍时长（秒）|
| delay_duration | int | 否 | 30 | 单次延时时长（秒）|
| max_delay_time | int | 否 | 180 | 最大延时时长（秒）|
| trigger_delay_before | int | 否 | 30 | 延时触发时间（秒）|

**响应示例**:
```json
{
  "id": 1,
  "product_id": 1,
  "start_price": 0,
  "increment": 10,
  "cap_price": 1000,
  "duration": 300,
  "delay_duration": 30,
  "max_delay_time": 180,
  "trigger_delay_before": 30,
  "created_at": "2026-05-21T18:00:00+08:00"
}
```

**说明**: 配置规则后会自动创建竞拍场次

**状态码**:
- `201`: 创建成功
- `400`: 参数错误
- `500`: 服务器错误

---

## 2. 竞拍服务 (Auction Service)

### 2.1 获取竞拍详情

**接口**: `GET /auctions/:id`

**响应示例**:
```json
{
  "id": 1,
  "product_id": 1,
  "status": 1,
  "current_price": 100.00,
  "winner_id": 2,
  "start_time": "2026-05-21T18:00:00+08:00",
  "end_time": "2026-05-21T18:05:00+08:00",
  "delay_used": 0,
  "created_at": "2026-05-21T18:00:00+08:00"
}
```

**竞拍状态**:
| 值 | 状态 | 说明 |
|----|------|------|
| 0 | pending | 待开始 |
| 1 | ongoing | 进行中 |
| 2 | delayed | 延时中 |
| 3 | ended | 已结束 |
| 4 | cancelled | 已取消 |

**状态码**:
- `200`: 成功
- `404`: 竞拍不存在

---

### 2.2 出价

**接口**: `POST /auctions/:id/bids`

**请求头**:
```
Content-Type: application/json
X-User-ID: 1001
```

**请求体**:
```json
{
  "amount": 150.00
}
```

**参数说明**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| amount | float | 是 | 出价金额（元）|

**响应示例**:

成功:
```json
{
  "success": true,
  "message": "出价成功",
  "current_price": 150.00,
  "rank": 1,
  "winner_id": 1001
}
```

失败:
```json
{
  "success": false,
  "message": "出价金额不足，最低出价为 110.00 元"
}
```

**出价规则**:
1. 出价金额必须 ≥ 当前价格 + 加价幅度
2. 达到封顶价自动成交
3. 使用分布式锁保证并发安全

**状态码**:
- `200`: 请求成功（业务成功或失败查看success字段）
- `400`: 参数错误
- `500`: 服务器错误

---

### 2.3 获取竞拍排名

**接口**: `GET /auctions/:id/ranking`

**响应示例**:
```json
{
  "items": [
    {
      "rank": 1,
      "user_id": 1001,
      "amount": 150.00,
      "bid_time": "2026-05-21T18:01:00+08:00"
    },
    {
      "rank": 2,
      "user_id": 1002,
      "amount": 140.00,
      "bid_time": "2026-05-21T18:00:30+08:00"
    }
  ]
}
```

**状态码**:
- `200`: 成功

---

### 2.4 取消竞拍

**接口**: `PUT /auctions/:id/cancel`

**说明**: 仅可取消未开始的竞拍

**响应示例**:
```json
{
  "code": 200,
  "message": "竞拍已取消"
}
```

**状态码**:
- `200`: 取消成功
- `400`: 竞拍已开始，无法取消
- `404`: 竞拍不存在

---

### 2.5 获取竞拍结果

**接口**: `GET /auctions/:id/result`

**响应示例**:
```json
{
  "auction_id": 1,
  "product_id": 1,
  "status": 3,
  "final_price": 150.00,
  "winner_id": 1001,
  "started_at": "2026-05-21T18:00:00+08:00",
  "ended_at": "2026-05-21T18:05:00+08:00",
  "delay_used": 0
}
```

**状态码**:
- `200`: 成功
- `404`: 竞拍不存在

---

## 3. WebSocket 服务

### 3.1 WebSocket 连接

**接口**: `ws://localhost:8083/ws`

**连接参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| auction_id | int | 是 | 竞拍ID |
| user_id | int | 否 | 用户ID（不提供则自动生成）|

**连接示例**:
```javascript
const ws = new WebSocket('ws://localhost:8083/ws?auction_id=1&user_id=1001');

ws.onopen = () => {
  console.log('WebSocket已连接');
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('收到消息:', data);
};
```

---

### 3.2 消息类型

#### 客户端 → 服务端

**心跳消息**:
```json
{
  "type": "ping"
}
```

**响应**:
```json
{
  "type": "pong",
  "timestamp": 1716288000000
}
```

---

#### 服务端 → 客户端

**1. 系统消息**:
```json
{
  "type": "system",
  "timestamp": 1716288000000,
  "data": {
    "message": "连接成功",
    "user_id": 1001,
    "auction_id": 1
  }
}
```

**2. 出价通知**:
```json
{
  "type": "bid_placed",
  "timestamp": 1716288000000,
  "data": {
    "user_id": 1001,
    "amount": 150.00,
    "current_price": 150.00,
    "bid_time": 1716288000000
  }
}
```

**3. 价格更新**:
```json
{
  "type": "price_update",
  "timestamp": 1716288000000,
  "data": {
    "user_id": 1001,
    "price": 150.00,
    "rank": 1,
    "auction_id": 1
  }
}
```

**4. 排名更新**:
```json
{
  "type": "rank_update",
  "timestamp": 1716288000000,
  "data": {
    "auction_id": 1,
    "ranking": [
      {
        "rank": 1,
        "user_id": 1001,
        "amount": 150.00
      }
    ]
  }
}
```

**5. 延时触发**:
```json
{
  "type": "delay_triggered",
  "timestamp": 1716288000000,
  "data": {
    "auction_id": 1,
    "delay_duration": 30,
    "new_end_time": 1716288300000,
    "remaining_delay": 150,
    "max_delay": 180
  }
}
```

**6. 竞拍结束**:
```json
{
  "type": "auction_ended",
  "timestamp": 1716288000000,
  "data": {
    "auction_id": 1,
    "winner_id": 1001,
    "final_price": 150.00,
    "end_time": 1716288000000
  }
}
```

**7. 时间同步**:
```json
{
  "type": "time_sync",
  "timestamp": 1716288000000,
  "data": {
    "server_time": 1716288000000,
    "end_time": 1716288300000
  }
}
```

---

## 4. 订单服务

### 4.1 获取订单列表

**接口**: `GET /orders`

**查询参数**:
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | int | 否 | 1 | 页码 |
| page_size | int | 否 | 20 | 每页数量 |

**响应示例**:
```json
{
  "items": [
    {
      "id": 1,
      "auction_id": 1,
      "product_id": 1,
      "winner_id": 1001,
      "final_price": 150.00,
      "status": 0,
      "created_at": "2026-05-21T18:05:00+08:00"
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1
}
```

**订单状态**:
| 值 | 状态 | 说明 |
|----|------|------|
| 0 | pending | 待支付 |
| 1 | paid | 已支付 |
| 2 | shipped | 已发货 |
| 3 | completed | 已完成 |

---

### 4.2 模拟支付

**接口**: `POST /orders/:id/pay`

**响应示例**:
```json
{
  "code": 200,
  "message": "支付成功"
}
```

**状态码**:
- `200`: 支付成功
- `404`: 订单不存在

---

## 5. 错误码说明

### 通用错误码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 201 | 创建成功 |
| 400 | 请求参数错误 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

### 业务错误消息

| 消息 | 原因 | 解决方案 |
|------|------|----------|
| "出价金额不足" | 出价金额低于最低要求 | 增加出价金额 |
| "竞拍已结束" | 竞拍已结束或取消 | 无法继续出价 |
| "竞拍不存在" | 竞拍ID无效 | 检查竞拍ID |
| "商品名称不能为空" | 创建商品时未提供名称 | 提供商品名称 |

---

## 6. 限流策略

| 服务 | 接口 | 限流策略 | QPS上限 |
|------|------|----------|---------|
| Gateway | 出价接口 | 令牌桶 | 1000/s |
| Gateway | 商品列表 | 滑动窗口 | 500/s |
| WebSocket | 连接数 | 连接数限制 | 1000/房间 |

---

## 7. 最佳实践

### 7.1 出价流程

1. 获取竞拍详情，确认状态为 `ongoing` 或 `delayed`
2. 计算最低出价金额 = 当前价格 + 加价幅度
3. 发送出价请求
4. 根据响应判断是否成功
5. 监听WebSocket获取实时更新

### 7.2 WebSocket使用建议

- 连接后立即发送心跳消息
- 监听所有消息类型，实时更新UI
- 断线后使用指数退避重连
- 使用服务端时间校准倒计时

### 7.3 性能优化建议

- 使用WebSocket推送代替轮询
- 实现消息节流，避免频繁更新UI
- 缓存竞拍详情，减少API调用
- 使用防抖处理用户快速点击

---

## 8. 测试用例

### 8.1 创建商品并开始竞拍

```bash
# 1. 创建商品
curl -X POST http://localhost:8080/api/v1/products \
  -H "Content-Type: application/json" \
  -d '{"name": "测试商品", "description": "测试描述"}'

# 2. 配置竞拍规则（会自动创建竞拍）
curl -X POST http://localhost:8080/api/v1/products/1/rules \
  -H "Content-Type: application/json" \
  -d '{"start_price": 0, "increment": 10, "duration": 300}'

# 3. 查看竞拍详情
curl http://localhost:8080/api/v1/auctions/1

# 4. 出价
curl -X POST http://localhost:8080/api/v1/auctions/1/bids \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 1" \
  -d '{"amount": 50}'
```

---

## 9. 联系方式

- **API问题**: 提交 GitHub Issue
- **文档改进**: 提交 Pull Request
- **技术支持**: 查看 `docs/deployment.md`
