# API Contracts: 后端API补充

**Feature**: `20260528-backend-api-seed`
**Date**: 2026-05-28

## Gateway路由补充

---

### 1. 订单发货

**Endpoint**: `PUT /api/v1/orders/:id/ship`
**Auth**: JWT + RequireMerchant/RequireAdmin
**Target**: Product Service

#### Request
```json
{
  "tracking_number": "SF1234567890",  // 可选，物流单号
  "shipping_company": "顺丰速运"       // 可选，物流公司
}
```

#### Response (200)
```json
{
  "code": 200,
  "data": {
    "id": 123,
    "status": 2,
    "shipped_at": "2026-05-28T10:00:00Z"
  },
  "message": "发货成功"
}
```

#### Error Responses
- 401: 未登录或权限不足
- 404: 订单不存在
- 400: 订单状态不允许发货

---

### 2. 用户订单历史

**Endpoint**: `GET /api/v1/orders/history`
**Auth**: JWT
**Target**: Product Service

#### Query Parameters
- `page`: 页码，默认1
- `page_size`: 每页数量，默认20

#### Response (200)
```json
{
  "code": 200,
  "data": {
    "list": [
      {
        "id": 123,
        "product_id": 1,
        "product_name": "明代青花瓷瓶",
        "final_price": 50000,
        "status": 3,
        "created_at": "2026-05-20T10:00:00Z"
      }
    ],
    "total": 5
  }
}
```

---

### 3. 管理端直播间列表

**Endpoint**: `GET /api/v1/admin/live-streams`
**Auth**: JWT + RequireAdmin
**Target**: Product Service

#### Query Parameters
- `page`: 页码
- `page_size`: 每页数量
- `status`: 状态筛选 (0/1)

#### Response (200)
```json
{
  "code": 200,
  "data": {
    "list": [
      {
        "id": 1,
        "name": "珠宝专场直播",
        "creator_id": 100,
        "creator_name": "张商家",
        "status": 1,
        "product_count": 5,
        "created_at": "2026-05-20T10:00:00Z"
      }
    ],
    "total": 10
  }
}
```

---

### 4. 直播间详情

**Endpoint**: `GET /api/v1/live-streams/:id`
**Auth**: 无
**Target**: Product Service

#### Response (200)
```json
{
  "code": 200,
  "data": {
    "id": 1,
    "name": "珠宝专场直播",
    "description": "高端珠宝竞拍专场",
    "cover_image": "https://...",
    "creator": {
      "id": 100,
      "name": "张商家"
    },
    "products": [
      {
        "id": 1,
        "name": "卡地亚钻石项链",
        "category": "珠宝名表"
      }
    ],
    "auctions": [
      {
        "id": 1,
        "product_id": 1,
        "status": 1,
        "current_price": 10000
      }
    ]
  }
}
```

---

## 类别管理API

---

### 5. 类别列表

**Endpoint**: `GET /api/v1/categories`
**Auth**: 无

#### Query Parameters
- `status`: 状态筛选 (可选)

#### Response (200)
```json
{
  "code": 200,
  "data": {
    "list": [
      {
        "id": 1,
        "name": "艺术收藏",
        "code": "art",
        "sort_order": 1,
        "status": 1
      }
    ],
    "total": 6
  }
}
```

---

### 6. 新增类别

**Endpoint**: `POST /api/v1/categories`
**Auth**: JWT + RequireAdmin

#### Request
```json
{
  "name": "艺术收藏",
  "code": "art",
  "description": "古董、字画、玉器等艺术品",
  "sort_order": 1
}
```

#### Response (201)
```json
{
  "code": 201,
  "data": {
    "id": 1,
    "name": "艺术收藏",
    "code": "art"
  },
  "message": "创建成功"
}
```

#### Error Responses
- 400: code已存在

---

### 7. 更新类别

**Endpoint**: `PUT /api/v1/categories/:id`
**Auth**: JWT + RequireAdmin

#### Request
```json
{
  "name": "艺术收藏品",
  "description": "更新后的描述"
}
```

#### Response (200)
```json
{
  "code": 200,
  "data": {
    "id": 1,
    "name": "艺术收藏品"
  },
  "message": "更新成功"
}
```

---

### 8. 删除类别

**Endpoint**: `DELETE /api/v1/categories/:id`
**Auth**: JWT + RequireAdmin

#### Response (200)
```json
{
  "code": 200,
  "message": "删除成功"
}
```

#### Error Responses
- 400: 该类别下有商品，无法删除