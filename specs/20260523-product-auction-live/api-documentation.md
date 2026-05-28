# API Documentation

**Base URL**: `http://localhost:8080/api/v1`
**Version**: 1.0.0
**Feature**: 20260523-product-auction-live

---

## Authentication

All API requests require JWT authentication unless specified otherwise.

**Header**: `Authorization: Bearer <token>`

---

## Endpoints

### 1. Product Management (Product Service)

#### POST /products/:id/publish
发布商品到直播间

**Permission**: 商家或管理员

**Request Body**:
```json
{
  "start_time": "2026-05-23T15:00:00Z",  // 可选，竞拍开始时间，默认30分钟后
  "rule_id": 123                          // 可选，竞拍规则ID
}
```

**Response**:
```json
{
  "code": 200,
  "message": "发布成功",
  "data": {
    "product": {
      "id": 1,
      "name": "稀有珠宝",
      "status": 1,  // 已发布
      "created_at": "2026-05-23T09:00:00Z"
    },
    "live_stream": {
      "id": 10,
      "name": "张三的直播间"
    }
  }
}
```

**Error Codes**:
- 400: 商品状态不正确
- 403: 权限不足
- 500: 服务器错误

---

#### POST /products/:id/unpublish
下架商品

**Permission**: 商家或管理员

**Request Body**:
```json
{
  "reason": "商品质量问题"  // 可选，下架原因
}
```

**Response**:
```json
{
  "code": 200,
  "message": "下架成功",
  "data": {
    "product_id": 1,
    "status": 2,  // 已下架
    "unpublished": true
  }
}
```

---

#### GET /products
获取商品列表

**Query Parameters**:
- `page`: 页码，默认1
- `page_size`: 每页数量，默认20
- `status`: 状态筛选（0=草稿, 1=已发布, 2=已下架）

**Response**:
```json
{
  "code": 200,
  "data": {
    "items": [
      {
        "id": 1,
        "name": "稀有珠宝",
        "description": "限量版珠宝",
        "status": 1,
        "created_at": "2026-05-23T09:00:00Z"
      }
    ],
    "total": 50,
    "page": 1,
    "page_size": 20
  }
}
```

---

### 2. Live Stream Management (Auction Service)

#### POST /live-streams/:id/follow
关注直播间

**Permission**: 所有登录用户

**Response**:
```json
{
  "code": 200,
  "message": "关注成功",
  "data": {
    "live_stream": {
      "id": 10,
      "name": "张三的直播间",
      "followers_count": 1250
    },
    "follow": {
      "id": 1001,
      "user_id": 5,
      "live_stream_id": 10,
      "notification_enabled": true,
      "created_at": "2026-05-23T09:30:00Z"
    }
  }
}
```

---

#### DELETE /live-streams/:id/follow
取消关注直播间

**Permission**: 所有登录用户

**Response**:
```json
{
  "code": 200,
  "message": "取消关注成功"
}
```

---

#### GET /live-streams/:id/followers/stats
获取直播间关注统计

**Permission**: 商家（自己的直播间）或管理员

**Response**:
```json
{
  "code": 200,
  "data": {
    "total_count": 1250,
    "new_today": 15,
    "new_this_week": 120,
    "new_this_month": 450,
    "active_last_7_days": 800,
    "active_last_30_days": 1000,
    "participated_count": 650
  }
}
```

---

#### GET /user/followed-live-streams
获取用户关注的直播间列表

**Permission**: 所有登录用户

**Query Parameters**:
- `page`: 页码，默认1
- `page_size`: 每页数量，默认20

**Response**:
```json
{
  "code": 200,
  "data": {
    "items": [
      {
        "id": 10,
        "name": "张三的直播间",
        "cover_image": "https://example.com/cover.jpg",
        "followers_count": 1250,
        "active_auctions": 3,
        "notification_enabled": true,
        "followed_at": "2026-05-23T09:30:00Z"
      }
    ],
    "total": 5,
    "page": 1,
    "page_size": 20
  }
}
```

---

### 3. Auction Management (Auction Service)

#### POST /auctions/:id/bids
用户出价竞拍

**Permission**: 所有登录用户

**Authentication**: 必须携带有效的JWT token，系统将从token中提取user_id

**Request Body**:
```json
{
  "amount": 150.00  // 出价金额，必须大于当前最高价+最小加价幅度
}
```

**Response (成功)**:
```json
{
  "code": 200,
  "message": "出价成功",
  "data": {
    "bid_id": 1001,
    "auction_id": 100,
    "user_id": 5,
    "amount": 150.00,
    "current_rank": 1,
    "bid_time": "2026-05-23T10:15:30Z"
  }
}
```

**Error Response (401 未认证)**:
```json
{
  "code": 401,
  "message": "未认证，请先登录"
}
```

**Error Response (400 出价金额不足)**:
```json
{
  "code": 400,
  "message": "出价金额不足，最小出价金额为155.00"
}
```

**Error Response (409 竞拍已结束)**:
```json
{
  "code": 409,
  "message": "竞拍已结束，无法出价"
}
```

**认证流程说明**：
1. 用户必须在请求Header中携带有效的JWT token：`Authorization: Bearer <token>`
2. 后端中间件验证token有效性，提取user_id并存入请求上下文
3. 出价处理程序从上下文获取user_id：`c.Get("user_id")`
4. 如果上下文中无user_id，返回401错误

**业务规则**：
- 出价金额必须 > 当前最高价 + 最小加价幅度
- 竞拍状态必须为"进行中"或"延时中"
- 竞拍未结束（end_time > now）
- 出价成功后自动更新竞拍排名
- 如果触发延时条件，自动延长竞拍时间
- 通过WebSocket实时推送给出价用户和其他参与者

---

#### GET /auctions/:id/ranking
获取竞拍排名

**Permission**: 所有人（包括未登录用户）

**Query Parameters**:
- `limit`: 返回数量，默认10

**Response**:
```json
{
  "code": 200,
  "data": {
    "items": [
      {
        "rank": 1,
        "user_id": 5,
        "amount": 150.00,
        "bid_time": "2026-05-23T10:15:30Z"
      },
      {
        "rank": 2,
        "user_id": 8,
        "amount": 145.00,
        "bid_time": "2026-05-23T10:14:20Z"
      }
    ]
  }
}
```

---

#### GET /auctions
获取竞拍列表

**Permission**: 商家或管理员

**Query Parameters**:
- `page`: 页码，默认1
- `page_size`: 每页数量，默认20
- `status`: 状态筛选（0=待开始, 1=进行中, 2=延时中, 3=已结束）
- `live_stream_id`: 按直播间ID筛选
- `live_stream_name`: 按直播间名称模糊搜索
- `search`: 关键词搜索

**Response (商家视角)**:
```json
{
  "code": 200,
  "data": {
    "items": [
      {
        "id": 100,
        "product_id": 1,
        "product_name": "稀有珠宝",
        "status": 1,
        "current_price": 150.00,
        "bid_count": 12,
        "start_time": "2026-05-23T10:00:00Z",
        "end_time": "2026-05-23T10:05:00Z"
      }
    ],
    "total": 10,
    "page": 1,
    "page_size": 20
  }
}
```

**Response (管理员视角)**:
```json
{
  "code": 200,
  "data": {
    "items": [
      {
        "id": 100,
        "product_id": 1,
        "product_name": "稀有珠宝",
        "live_stream_id": 10,
        "live_stream_name": "张三的直播间",
        "creator_id": 5,
        "creator_name": "张三",
        "status": 1,
        "current_price": 150.00,
        "bid_count": 12,
        "start_time": "2026-05-23T10:00:00Z",
        "end_time": "2026-05-23T10:05:00Z",
        "remaining_time": "3分25秒",
        "winner_name": "用户A"
      }
    ],
    "total": 50,
    "page": 1,
    "page_size": 20
  }
}
```

---

### 4. Notification API (Internal)

#### POST /internal/notifications/batch-push
批量推送通知（内部API）

**Permission**: 内部服务调用

**Request Body**:
```json
{
  "live_stream_id": 10,
  "type": "new_product",
  "title": "新商品上架",
  "content": "直播间【张三的直播间】发布了新商品【稀有珠宝】，快来参与竞拍吧！",
  "data": {
    "product_id": 1,
    "auction_id": 100
  }
}
```

**Response**:
```json
{
  "code": 200,
  "message": "推送任务已创建",
  "data": {
    "task_id": "task_20260523_123456",
    "target_count": 1250,
    "estimated_duration": "5分钟"
  }
}
```

---

## Error Codes

| Code | Description |
|------|-------------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 409 | 资源冲突（如重复关注） |
| 500 | 服务器内部错误 |

---

## Permission Matrix

| API Endpoint | User (Role=0) | Merchant (Role=1) | Admin (Role=2) |
|--------------|---------------|-------------------|----------------|
| POST /products/:id/publish | ❌ | ✅ (自己的商品) | ✅ |
| POST /products/:id/unpublish | ❌ | ✅ (自己的商品) | ✅ |
| POST /live-streams/:id/follow | ✅ | ✅ | ✅ |
| DELETE /live-streams/:id/follow | ✅ | ✅ | ✅ |
| GET /live-streams/:id/followers/stats | ❌ | ✅ (自己的直播间) | ✅ |
| GET /auctions | ❌ | ✅ (自己的直播间) | ✅ |
| POST /auctions/:id/bids | ✅ | ✅ | ✅ |
| GET /auctions/:id/ranking | ✅ | ✅ | ✅ |

<!-- clarify: 2026-05-23 — Added bid and ranking endpoints with authentication requirements -->

---

## Performance Requirements

| API Endpoint | Response Time | Notes |
|--------------|---------------|-------|
| POST /products/:id/publish | < 500ms | 不包含推送时间 |
| POST /products/:id/unpublish | < 1s | 包含取消竞拍和发送通知 |
| GET /auctions | < 1s | 包含搜索和筛选 |
| POST /internal/notifications/batch-push | < 100ms | 仅创建推送任务 |

---

## Rate Limits

- 默认：100 requests/minute/user
- 批量推送：10 requests/minute/service

---

## Versioning

API version is included in the URL path: `/api/v1/`

Breaking changes will result in a new version number.

---

## Testing

### Using cURL

```bash
# 登录获取token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"admin123"}'

# 发布商品
curl -X POST http://localhost:8080/api/v1/products/1/publish \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"start_time":"2026-05-23T15:00:00Z"}'

# 关注直播间
curl -X POST http://localhost:8080/api/v1/live-streams/1/follow \
  -H "Authorization: Bearer <token>"

# 用户出价（必须登录）
curl -X POST http://localhost:8080/api/v1/auctions/100/bids \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"amount": 150.00}'

# 获取竞拍排名（无需登录）
curl -X GET "http://localhost:8080/api/v1/auctions/100/ranking?limit=10"
```

<!-- clarify: 2026-05-23 — Added bid API test example with authentication -->

### Using Postman

Import collection from: `postman_collection.json` (待生成)

---

## Changelog

### v1.1.0 (2026-05-23)
- Added user bid endpoint with JWT authentication requirement
- Added auction ranking endpoint
- Updated permission matrix to include user bid access
- Added bid API test examples

### v1.0.0 (2026-05-23)
- Initial API release
- Product management endpoints
- Live stream follow endpoints
- Auction management enhancements
- Notification batch push API

---

**Generated**: 2026-05-23
**Status**: Specification Complete
**Implementation**: 45% Complete
**Last Updated**: 2026-05-23 - Added user bid authentication requirements
